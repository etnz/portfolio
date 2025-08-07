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
	"strings"

	"github.com/etnz/portfolio/date"
)

const attrOn = "on"
const marketDataFilesGlob = "[0-9][0-9][0-9][0-9].jsonl"
const definitionFilename = "definition.jsonl"

// This file contains code to persist market data in a folder, in a way that is still human-readable and git-friendly.
// the main goal for such market data is to live on a private github repo.
//
// The overall strategy to Encode/Decode market data is as follow:
//   Decode: read all files with a glob into a list of lines (with metadata like filename and line number)
//         Then parse each json line and add append it to the database.
//
//   Encode: create a list of tickers in alphabetical order.
//            Then scan all days for a ticker's value, and append them to the list of structured lines, including the filename.
//            Then generate each file.
//            Then using the same glob, create the list of all existing files on the disk, and compute which one is to be deleted.

// decodeDefinition parses a single file containing the securities definition.
// filename is for error message only.
func (m *MarketData) decodeDefinition(filename string, r io.Reader) error {
	// to parse a json, we use a dedicated local struct with tag annotation.

	// jsecurity is the object read from the file using json parser.
	type jsecurity struct {
		Ticker   string `json:"ticker"`
		ID       string `json:"id"`
		Currency string `json:"currency"`
		//more to come when the security definition grows.
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
			return fmt.Errorf("format error in %q: ticker %q is already defined", filename, js.Ticker)
		}
		sec := &Security{
			ticker:   js.Ticker,
			id:       ID(js.ID),
			currency: js.Currency,
		}
		m.Add(sec)
	}
	return nil
}

// fileLine structures a line from a collection of files as the persistence layer represent them.
type fileLine struct {
	filename string
	i        int
	txt      string
}

// decodeLines read all lines from a set of files and return them in list of structured lines.
func decodeLines(filenames ...string) (list []fileLine, err error) {
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

// decodeLine decodes a single line from the database persisted files.
func decodeLine(m *MarketData, l fileLine) error {

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

// DecodeMarketData reads a folder containing securities definition and prices, and returns a MarketData object.
func DecodeMarketData(folder string) (*MarketData, error) {
	// Creates an empty database.
	m := NewMarketData()

	// strategy: reads the metadata file containing securities definition and ticker, then uses it to load prices.
	// then read all json files and break it into lines, and load them individually.

	definitionFile := filepath.Join(folder, definitionFilename)
	f, err := os.Open(definitionFile)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("does not exists", err)
		}
		return nil, fmt.Errorf("load error: cannot open market definition file %q: %w", definitionFile, err)
	}
	defer f.Close()

	if err := m.decodeDefinition(definitionFile, f); err != nil {
		return nil, fmt.Errorf("load error: cannot read market definition file: %w", err)
	}

	// Use global to find all the files that are part of the db.
	filenames, err := filepath.Glob(filepath.Join(folder, marketDataFilesGlob))
	if err != nil {
		return nil, fmt.Errorf("load error: cannot scan folder %q for market data files: %w", folder, err)
	}

	lines, err := decodeLines(filenames...)
	if err != nil {
		return nil, err // err is already a package error
	}

	for _, line := range lines {

		if err := decodeLine(m, line); err != nil {
			return nil, err
		}

	}
	return m, nil
}

// Persist section.

// encodeDefinition encodes the securities definition into a jsonl stream.
func encodeDefinition(w io.Writer, m *MarketData) error {
	// jsecurity is the object to write to the file using json parser.
	type jsecurity struct {
		Ticker   string `json:"ticker"`
		ID       string `json:"id"`
		Currency string `json:"currency"`
		//more to come when the security definition grows.
	}

	for _, sec := range m.securities {
		js := jsecurity{
			Ticker:   sec.Ticker(),
			ID:       string(sec.ID()),
			Currency: sec.currency,
		}

		data, err := json.Marshal(js)
		if err != nil {
			return fmt.Errorf("persist error: cannot marshal security %q: %w", sec.ticker, err)
		}

		if _, err := w.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("persist error: cannot write to file: %w", err)
		}
	}
	return nil
}

// encodeLine persists a single line in a security jsonl file.
// Returns bare io errors.
func encodeLine(w io.Writer, day date.Date, tickers []string, values []float64) error {
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

// EncodeMarketData encodes the market data into a folder, creating a definition file and a set of jsonl files for each year.
func EncodeMarketData(folder string, m *MarketData) error {

	// we first generate the security price values into this list of structured items.
	type line struct {
		filename string
		day      date.Date
		tickers  []string
		prices   []float64
	}
	lines := make([]line, 0, 365*100) // hunderd years should be enough

	// The m.securities slice is already sorted, so we can use it directly.
	histories := make([]date.History[float64], 0, len(m.securities))
	for _, sec := range m.securities {
		prices, exists := m.prices[sec.ID()]
		if !exists {
			return fmt.Errorf("invalid market data: security %q has no prices", sec.Ticker())
		}
		histories = append(histories, *prices)
	}

	// Persist the definition file.
	definitionFile := filepath.Join(folder, definitionFilename)
	f, err := os.Create(definitionFile)
	if err != nil {
		return fmt.Errorf("persist error: cannot create file %q: %w", definitionFile, err)
	}
	defer f.Close()
	log.Printf("create-market-definition-file name=%q", definitionFile)

	if err := encodeDefinition(f, m); err != nil {
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
		for _, sec := range m.securities {
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
			createdFiles[currentFilename] = struct{}{} // Append this file to the list of created ones.
			defer currentFile.Close()
			log.Printf("create-market-data-file name=%q", currentFilename)
		}

		// Write line to currentFile.
		if err := encodeLine(currentFile, l.day, l.tickers, l.prices); err != nil {
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

// DecodeLedger decodes transactions from a stream of JSONL data from an io.Reader,
// decodes each line into the appropriate transaction struct, and returns a sorted Ledger.
func DecodeLedger(r io.Reader) (*Ledger, error) {
	ledger := NewLedger()
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		lineBytes := scanner.Bytes()
		if len(lineBytes) == 0 {
			continue // Skip empty lines
		}

		var identifier struct {
			Command CommandType `json:"command"`
		}
		if err := json.Unmarshal(lineBytes, &identifier); err != nil {
			return nil, fmt.Errorf("could not identify command in line %q: %w", string(lineBytes), err)
		}

		var decodedTx Transaction
		var err error

		switch identifier.Command {
		case CmdBuy:
			var tx Buy
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdSell:
			var tx Sell
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdDividend:
			var tx Dividend
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdDeposit:
			var tx Deposit
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdWithdraw:
			var tx Withdraw
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		case CmdConvert:
			var tx Convert
			err = json.Unmarshal(lineBytes, &tx)
			decodedTx = tx
		default:
			err = fmt.Errorf("unknown transaction command: %q", identifier.Command)
		}

		if err != nil {
			return nil, err
		}
		ledger.transactions = append(ledger.transactions, decodedTx)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading from input: %w", err)
	}

	// Perform a stable sort on the ledger based on the transaction date.
	ledger.stableSort()

	return ledger, nil
}

// EncodeTransaction marshals a single transaction to JSON and writes it to the
// writer, followed by a newline, in JSONL format.
func EncodeTransaction(w io.Writer, tx Transaction) error {
	jsonData, err := json.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal transaction: %w", err)
	}

	// Write the JSON data followed by a newline to create the JSONL format.
	if _, err := w.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write transaction: %w", err)
	}
	return nil
}

// EncodeLedger reorders transactions by date and persists them to an io.Writer in JSONL format.
// The sort is stable, meaning transactions on the same day maintain their original relative order.
func EncodeLedger(w io.Writer, ledger *Ledger) error {
	// Perform a stable sort on the ledger based on the transaction date to ensure order.
	ledger.stableSort()

	// 2. Iterate through the sorted transactions and write each one as a JSON line.
	for _, tx := range ledger.transactions {
		if err := EncodeTransaction(w, tx); err != nil {
			return err
		}
	}

	return nil
}
