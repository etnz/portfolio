package portfolio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const attrOn = "on"
const marketDataFilesGlob = "[0-9][0-9][0-9][0-9].jsonl"

// This file contains code to persist market data in a folder, in a way that is still human-readable and git-friendly.
// the main goal for such market data is to live on a private github repo.
//
// The overall strategy to Encode/Decode market data is as follows:
//   Decode: read all files with a glob into a list of lines (with metadata like filename and line number)
//         Then parse each json line and add append it to the database.
//
//   Encode: create a list of tickers in alphabetical order.
//            Then scan all days for a ticker's value, and append them to the list of structured lines, including the filename.
//            Then generate each file.
//            Then using the same glob, create the list of all existing files on the disk, and compute which one is to be deleted.

// decodeSecurities parses a single file containing the securities definition.
// filename is for error message only.
func (m *MarketData) decodeSecurities(filename string, r io.Reader) error {
	// to parse a json, we use a dedicated local struct with tag annotation.

	// jsecurity is the object read from the file using json parser.
	type jsecurity struct {
		Ticker   string  `json:"ticker"`
		ID       string  `json:"id"`
		Currency string  `json:"currency"`
		Splits   []Split `json:"splits,omitempty"`
	}

	// The definition file is a JSONL file, one security per line.
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}

		var js jsecurity
		if err := json.Unmarshal(line, &js); err != nil {
			return fmt.Errorf("format error in %q on line %q: %w", filename, string(line), err)
		}

		if m.Has(js.Ticker) {
			log.Printf("format error in %q: ticker %q is already defined", filename, js.Ticker)
			continue
		}
		sec := Security{
			ticker:   js.Ticker,
			id:       ID(js.ID),
			currency: js.Currency,
		}
		m.Add(sec)
		for _, split := range js.Splits {
			m.AddSplit(sec.ID(), split)
		}
	}
	return nil
}

// fileLine structures a line from a collection of files as the persistence layer represent them.
type fileLine struct {
	filename string
	i        int
	txt      string
}

// loadLines read all lines from a set of files and return them in list of structured lines.
func loadLines(filenames ...string) (list []fileLine, err error) {
	list = make([]fileLine, 0, 100000)
	for _, filename := range filenames {
		i := 0
		r, err := os.Open(filename)
		if err != nil {
			return nil, fmt.Errorf("cannot open %q for reading: %w", filename, err) //
		}
		scanner := bufio.NewScanner(r)
		for scanner.Scan() {
			i++
			txt := scanner.Text()
			list = append(list, fileLine{filename, i, txt})
		}
	}
	return list, nil
}

// decodeDailyPrices decodes a single line from the database persisted files.
func decodeDailyPrices(m *MarketData, l fileLine) error {

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

	on, err := ParseDate(jstring)
	if err != nil {
		return fmt.Errorf("parse error %s:%v: property %q must be a valid date: %w", l.filename, l.i, attrOn, err)
	}

	// Read all other attributes as (ticker, price) pairs.
	for ticker, price := range jobj {
		if ticker == attrOn { // skip this one, we read it as the timestamp.
			// reserved word for timestamp
			continue
		}

		p, ok := price.(float64)
		if !ok {
			return fmt.Errorf("parse error %s:%v: property %q must be of type 'number'", l.filename, l.i, ticker)
		}

		id, exists := m.tickers[ticker]
		if !exists {
			return fmt.Errorf("parse error %s:%v: property %q must be an existing ticker", l.filename, l.i, ticker)
		}

		// Entry is valid add it to the database.
		m.Append(id, on, p)
	}
	return nil
}

// DecodeMarketData reads a folder containing securities definitions and prices, and returns a MarketData object.
func DecodeMarketData(marketFile string) (*MarketData, error) {
	folder := filepath.Dir(marketFile)
	// Creates an empty database.
	m := NewMarketData()

	// strategy: reads the metadata file containing securities definition and ticker, then uses it to load prices.
	// then read all json files and break it into lines, and load them individually.

	f, err := os.Open(marketFile)
	if err != nil {
		if os.IsNotExist(err) {
			return m, nil // Empty database.
		}
		return nil, fmt.Errorf("load error: cannot open market definition file %q: %w", marketFile, err)
	}
	defer f.Close()

	if err := m.decodeSecurities(marketFile, f); err != nil {
		return nil, fmt.Errorf("load error: cannot read market definition file: %w", err)
	}

	// Use global to find all the files that are part of the db.
	filenames, err := filepath.Glob(filepath.Join(folder, marketDataFilesGlob))
	if err != nil {
		return nil, fmt.Errorf("load error: cannot scan folder %q for market data files: %w", folder, err)
	}

	lines, err := loadLines(filenames...)
	if err != nil {
		return nil, err // err is already a package error
	}

	for _, line := range lines {

		if err := decodeDailyPrices(m, line); err != nil {
			return nil, err
		}

	}
	return m, nil
}

// Persist section.

// encodeSecurities encodes the securities definition into a jsonl stream.
func encodeSecurities(w io.Writer, m *MarketData) error {
	// jsecurity is the object to write to the file using json parser.
	type jsecurity struct {
		Ticker   string  `json:"ticker"`
		ID       string  `json:"id"`
		Currency string  `json:"currency"`
		Splits   []Split `json:"splits,omitempty"`
	}

	// Collect securities and sort them by ticker for stable output.
	var sortedSecurities []Security
	for _, sec := range m.securities {
		sortedSecurities = append(sortedSecurities, sec)
	}
	sort.Slice(sortedSecurities, func(i, j int) bool {
		return sortedSecurities[i].Ticker() < sortedSecurities[j].Ticker()
	})

	for _, sec := range sortedSecurities {
		js := jsecurity{
			Ticker:   sec.Ticker(),
			ID:       string(sec.ID()),
			Currency: sec.currency,
			Splits:   m.Splits(sec.ID()),
		}

		data, err := json.Marshal(js)
		if err != nil {
			return fmt.Errorf("persist error: cannot marshal security %q: %w", sec.Ticker(), err)
		}

		if _, err := w.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("persist error: cannot write to file: %w", err)
		}
	}
	return nil
}

// encodeDailyPrices persists a single line in a security jsonl file.
// Returns bare io errors.
func encodeDailyPrices(w io.Writer, day Date, tickers []string, values []float64) error {
	var jw jsonObjectWriter
	jw.Append(attrOn, day.String())

	// Write all (ticker,price) pairs
	for i, ticker := range tickers {
		price := values[i]

		// Skip nans. json does not support NaN. We could have uses null, but we have already checked that value existed.
		if math.IsNaN(price) {
			continue
		}
		jw.Append(ticker, price)
	}

	b, err := jw.MarshalJSON()
	if err != nil {
		return err
	}

	if _, err := w.Write(append(b, '\n')); err != nil {
		return err
	}

	return nil
}

// EncodeMarketData encodes the market data into a folder, creating a definition file and a set of JSONL files for each year.
func EncodeMarketData(definitionFile string, m *MarketData) error {

	// we first generate the security price values into this list of structured items.
	type line struct {
		filename string
		day      Date
		tickers  []string
		prices   []float64
	}
	lines := make([]line, 0, 365*100) // hunderd years should be enough

	// Collect securities and sort them by ticker for stable output.
	var sortedSecurities []Security
	for _, sec := range m.securities {
		sortedSecurities = append(sortedSecurities, sec)
	}
	sort.Slice(sortedSecurities, func(i, j int) bool {
		return sortedSecurities[i].Ticker() < sortedSecurities[j].Ticker()
	})

	histories := make([]History[float64], 0, len(sortedSecurities))
	for _, sec := range sortedSecurities {
		prices, exists := m.prices[sec.ID()]
		if !exists {
			return fmt.Errorf("invalid market data: security %q has no prices", sec.Ticker())
		}
		histories = append(histories, *prices)
	}

	// Persist the definition file.
	folder := filepath.Dir(definitionFile)
	f, err := os.Create(definitionFile)
	if err != nil {
		return fmt.Errorf("persist error: cannot create file %q: %w", definitionFile, err)
	}
	defer f.Close()
	log.Printf("create-market-definition-file name=%q", definitionFile)

	if err := encodeSecurities(f, m); err != nil {
		return err
	}
	// Add a trailing line at the end of the file.
	if _, err := fmt.Fprintln(f); err != nil {
		return fmt.Errorf("persist error: cannot write to file: %w", err)
	}

	// Scan the database and fill the 'lines' list of structured lines.

	for day := range Iterate(histories...) {
		// Init the line with current day, and a file name based on the year.
		l := line{
			day:      day,
			filename: filepath.Join(folder, fmt.Sprintf("%d.jsonl", day.Year())),
		}
		// Append tickers that have values.
		for _, sec := range sortedSecurities {
			if val, ok := m.read(sec.ID(), day); ok {
				l.tickers = append(l.tickers, sec.Ticker())
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
			createdFiles[currentFilename] = struct{}{}
			defer currentFile.Close()
			log.Printf("create-market-data-file name=%q", currentFilename)
		}

		// Write line to currentFile.
		if err := encodeDailyPrices(currentFile, l.day, l.tickers, l.prices); err != nil {
			return fmt.Errorf("persist error: write error on file %q: %w", currentFilename, err)
		}
	}

	// Delete extraneous files.

	filenames, err := filepath.Glob(filepath.Join(folder, marketDataFilesGlob))
	if err != nil {
		return fmt.Errorf("persist error: cannot scan folder %q for market data files to be deleted: %w", folder, err)
	}
	for _, filename := range filenames {
		if _, ok := createdFiles[filename]; ok {
			continue // skip created ones
		}
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("persist error: cannot delete file %q: %w", filename, err)
		}
		log.Printf("delete-market-data-file name=%q", filename)
	}
	return nil
}
