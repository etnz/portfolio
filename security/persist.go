package security

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/etnz/porfolio/date"
)

const attrOn = "on"
const securityFilesGlob = "[0-9][0-9][0-9][0-9].jsonl"
const definitionFilename = "definition.json"

// this file contains code to persist the database in a folder, in a way that is still human readable, and yet git friendly.
// the main goal for such a database is to live on a private github repo.
//
// The overall strategy to Load and Persist securities is as follow:
//   Load: read all files with a glob into a list of lines (with metadata like filename and line number)
//         Then parse each json line and add append it to the database.
//
//   Persist: create a list of tickers in alphabetical order
//            Then scan all days for ticker's value, and append them to the list of structured lines, including the filename.
//            Then generate each file
//            Then using the same glob create the list of all existing files on the disk, and compute which one is to be deleted.

// loadDefinition parses a single file containing the securities definition.
// filename is for error message only.
func (db *DB) loadDefinition(filename string, r io.Reader) error {
	// to parse a json, we use a dedicated local struct with tag annotation.

	// jsecurity is the object read from the file using json parser.
	type jsecurity struct {
		ID string `json:"id"`
		//more to come when the security definition grows.
	}

	// The top struct of the file format is a map of ticker to jsecurity objects.
	jsecurities := make(map[string]*jsecurity)

	if err := json.NewDecoder(r).Decode(&jsecurities); err != nil {
		return fmt.Errorf("format error %q: %w", filename, err)
	}

	// Now load the struct we just read into the database.
	for ticker, js := range jsecurities {
		if db.Has(ticker) {
			return fmt.Errorf("format error %q: ticker %q is already defined", filename, ticker)
		}
		// Create the real security object from the json proxy.
		db.content[ticker] = &Security{
			id: ID(js.ID),
		}
	}
	return nil
}

// line structures a line from a collection of files as the persistence layer represent them.
type line struct {
	filename string
	i        int
	txt      string
}

// readLines read all lines from a set of files and return them in list of structured lines.
func readLines(filenames ...string) (list []line, err error) {
	list = make([]line, 0, 100000)
	for _, filename := range filenames {
		i := 0
		r, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("cannot open %q for reading: %w", filename, err)
		}
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			i++
			txt := scanner.Text()
			list = append(list, line{filename, i, txt})
		}
	}
	return list, nil
}

// load a single line from the database persisted files.
func (db *DB) loadLine(l line) error {

	// Start simply ignoring empty lines.
	if strings.TrimSpace(l.txt) == "" {
		return nil
	}

	// Parse the line as json
	jobj := make(map[string]any)
	if err := json.Unmarshal([]byte(l.txt), &jobj); err != nil {
		return fmt.Errorf("parse error %s:%v: not a correct json: %w", l.filename, l.i, err)
	}

	// Read the timestamp
	jvalue, ok := jobj[attrOn]
	if !ok {
		return fmt.Errorf("parse error %s:%v: missing the property %q with a date", l.filename, l.i, attrOn)
	}
	jstring, ok := jvalue.(string)
	if !ok {
		return fmt.Errorf("parse error %s:%v: property %q must be of type 'string'", l.filename, l.i, attrOn)
	}

	on, err := date.Parse(jstring)
	if err != nil {
		return fmt.Errorf("parse error %s:%v: property %q must be a valid date: %w", l.filename, l.i, attrOn, err)
	}

	// Read all other attributes as (ticker,price) pairs.
	for ticker, price := range jobj {
		if ticker == attrOn { // skip this one, we read it as the timestamp.
			// reserved word for timestamp
			continue
		}

		if !db.Has(ticker) {
			return fmt.Errorf("parse error %s:%v: property %q must be an existing ticker", l.filename, l.i, ticker)
		}

		p, ok := price.(float64)
		if !ok {
			return fmt.Errorf("parse error %s:%v: property %q must be of type 'number'", l.filename, l.i, ticker)
		}
		// Entry is valid add it to the database.
		db.content[ticker].prices.Append(on, p)
	}
	return nil
}

// Load a database from its folder.
func Load(folder string) (*DB, error) {
	// Creates an empty database.
	db := NewDB()

	// strategy: reads the metadata file containing securities definition and ticker, then use it to load prices.
	// then read all json files and break it into lines, and load them individually.

	definitionFile := filepath.Join(folder, definitionFilename)
	f, err := os.Open(definitionFile)
	if err != nil {
		return nil, fmt.Errorf("load error: cannot open securities file %q: %w", definitionFile, err)
	}
	defer f.Close()

	if err := db.loadDefinition(definitionFile, f); err != nil {
		return nil, fmt.Errorf("load error: cannot read securities definition file: %w", err)
	}

	// Use global to find all the files that are part of the db.
	filenames, err := filepath.Glob(filepath.Join(folder, securityFilesGlob))
	if err != nil {
		return nil, fmt.Errorf("load error: cannot scan folder %q for security files: %w", folder, err)
	}

	lines, err := readLines(filenames...)
	if err != nil {
		return nil, err // err is already a package error
	}

	for _, line := range lines {
		if err := db.loadLine(line); err != nil {
			return nil, err
		}
	}
	return db, nil
}

// Persist section.

func persistSecurity(w io.Writer, sec *Security) error {
	// jsecurity is the object read from the file using json parser.
	type jsecurity struct {
		ID string `json:"id"`
		//more to come when the security definition grows.
	}
	js := jsecurity{
		ID: string(sec.id),
	}
	data, err := json.Marshal(js)
	if err != nil {
		return fmt.Errorf("persist error: cannot persist Security %q definition %w", sec.ticker, err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("persist error: cannot write to file: %w", err)
	}

	return nil
}

func (db *DB) persistDefinition(w io.Writer, tickers []string) error {
	// We cannot use Go standard serialisation as the definition file wouldn't be stable. (it contains a map)
	if _, err := fmt.Fprint(w, "{\n    "); err != nil {
		return fmt.Errorf("persist error: cannot write to file: %w", err)
	}

	for i, ticker := range tickers {
		sec, exists := db.content[ticker]
		if !exists {
			return fmt.Errorf("persist error: unknown ticker %q", ticker)
		}

		data, err := json.Marshal(ticker)
		if err != nil {
			return fmt.Errorf("persist error: invalid ticker %q: %w", ticker, err)
		}
		if _, err := w.Write(data); err != nil {
			return fmt.Errorf("persist error: cannot write to file: %w", err)
		}

		if _, err := fmt.Fprint(w, ":"); err != nil {
			return fmt.Errorf("persist error: cannot write to file: %w", err)
		}

		if err := persistSecurity(w, sec); err != nil {
			return err
		}

		if i != len(tickers)-1 { // not last
			if _, err := fmt.Fprint(w, ",\n    "); err != nil {
				return fmt.Errorf("persist error: cannot write to file: %w", err)
			}
		}
	}

	if _, err := fmt.Fprint(w, "\n}"); err != nil {
		return fmt.Errorf("persist error: cannot write to file: %w", err)
	}
	return nil
}

// PersistLine persist a single line in a security jsonl file.
// Returns bare io errors.
func persistLine(w io.Writer, day date.Date, tickers []string, values []float64) error {
	// json encoder cannot be used as it would require a map, and map order is not guaranteed.
	// Instead fine grained formatting is done.

	if _, err := fmt.Fprintf(w, "{ %q:%q", attrOn, day.String()); err != nil {
		return err
	}
	// Write all (ticker,price) pairs
	for i, ticker := range tickers {
		price := values[i]

		// Skip nans. json does not support NaN. We could have uses null, but we have already checked that value existed.
		if math.IsNaN(price) {
			continue
		}

		if _, err := fmt.Fprintf(w, ", %q:%v", ticker, price); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w, "}"); err != nil {
		return err
	}
	return nil
}

func (db *DB) Persist(folder string) error {

	// we first generate the security price values into this list of structured items.
	type line struct {
		filename string
		day      date.Date
		tickers  []string
		prices   []float64
	}
	lines := make([]line, 0, 365*100) // hunderd years should be enough

	// Start creating the list of tickers, in alphabetical order.
	tickers := make([]string, 0, len(db.content))
	histories := make([]date.History[float64], 0, len(db.content))
	for ticker, sec := range db.content {
		tickers = append(tickers, ticker)
		histories = append(histories, sec.prices)
	}
	slices.Sort(tickers)

	// Persist the definition file.
	definitionFile := filepath.Join(folder, definitionFilename)
	f, err := os.Create(definitionFile)
	if err != nil {
		return fmt.Errorf("persist error: cannot create file %q: %w", definitionFile, err)
	}
	defer f.Close()
	log.Printf("create-definition-file name=%q", definitionFile)

	if err := db.persistDefinition(f, tickers); err != nil {
		return err
	}
	// Add a trailing line at the end of the file.
	if _, err := fmt.Fprintln(f); err != nil {
		return fmt.Errorf("persist error: cannot write to file: %w", err)
	}

	// Scan the database and fill the 'lines' list of structured lines.

	for day := range date.Iterate(histories...) {
		// Init the line with current day, and a file name based on the year.
		l := line{
			day:      day,
			filename: filepath.Join(folder, fmt.Sprintf("%v.jsonl", day.Year())),
		}
		// Append tickers that have values.
		for _, ticker := range tickers {
			if val, ok := db.read(ticker, day); ok {
				l.tickers = append(l.tickers, ticker)
				l.prices = append(l.prices, val)
			}
		}
		lines = append(lines, l)
	}

	// Persist all lines into their corresponding files.

	var currentFile *os.File
	var currentFilename string
	var createdFiles = make(map[string]struct{})
	for _, l := range lines {
		// Check wether we should switch to a new file
		if currentFilename != l.filename {
			currentFilename = l.filename
			var err error
			currentFile, err = os.Create(currentFilename)
			if err != nil {
				return fmt.Errorf("persist error: cannot create file %q: %w", currentFilename, err)
			}
			createdFiles[currentFilename] = struct{}{} // Append this file to the list of created ones.
			defer currentFile.Close()
			log.Printf("create-security-file name=%q", currentFilename)
		}

		// Write line to currentFile.
		if err := persistLine(currentFile, l.day, l.tickers, l.prices); err != nil {
			return fmt.Errorf("persist error: write error on file %q: %w", currentFilename, err)
		}
	}

	// Delete extraneous files.

	filenames, err := filepath.Glob(filepath.Join(folder, securityFilesGlob))
	if err != nil {
		return fmt.Errorf("persist error: cannot scan folder %q for security files to be deleted: %w", folder, err)
	}
	for _, filename := range filenames {
		if _, ok := createdFiles[filename]; ok {
			continue // skip created ones
		}
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("persist error: cannot delete %q file: %w", filename, err)
		}
		log.Printf("delete-security-file name=%q", currentFilename)
	}
	return nil
}
