package security

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/etnz/portfolio/date"
)

// this file contains functions to handle the import/export format.
// It should remain human readable, single file and be easy to merge into a database.

// Import securities from 'r' in the import/export format.
//
// The import format is a json file.
//
// The file contains a single json object whose property names are security tickers and values are securities themselves.
//
// A security is a single json object whose property 'id' contains the security ID as string, and property 'history' contains a single json object representing the security history.
//
// The security history is represented as a single json object whose properties are date.Date parseable by [date] package, and value are the security price as a number.
func (s *Securities) Import(r io.Reader) error {

	// the readable version of the format is can be summarized by a few types.

	type jsecurity struct {
		ID      string             `json:"id"`
		History map[string]float64 `json:"history"`
	}
	content := make(map[string]jsecurity)
	if err := json.NewDecoder(r).Decode(&content); err != nil {
		return fmt.Errorf("cannot parse file for Security import format: %v", err)
	}

	// Check that tickers is not already present in the database.
	var tickers []string
	var dateErrors []error
	for ticker, js := range content {
		if _, ok := s.content[ticker]; ok {
			tickers = append(tickers, ticker)
		}
		for day := range js.History {
			if _, err := date.Parse(day); err != nil {
				dateErrors = append(dateErrors, fmt.Errorf("invalid date in %q history: %w", ticker, err))
			}
		}
	}
	if len(tickers) == 1 {
		return fmt.Errorf("ticker %v is already present in the database", tickers)
	}
	if len(tickers) > 1 {
		return fmt.Errorf("%v tickers %v are already present in the database", len(tickers), tickers)
	}
	if len(dateErrors) == 1 {
		return dateErrors[0]
	}
	if len(dateErrors) > 1 {
		return fmt.Errorf("errors parsing dates: %v", dateErrors)
	}

	// Append securities for each ticker
	for ticker, js := range content {
		// Create the security.
		sec := &Security{
			id: ID(js.ID),
		}

		// fill the security from json
		for day, value := range js.History {
			// error has been checked before
			d, _ := date.Parse(day)
			sec.prices.Append(d, value)
		}
		s.content[ticker] = sec
	}
	return nil
}

// Export securities in database to 'w' in the import/export format.
//
// The format is a json file.
//
// The file contains a single json object whose property names are security tickers and values are securities themselves.
//
// A security is a single json object whose property 'id' contains the security ID as string, and property 'history' contains a single json object representing the security history.
//
// The security history is represented as a single json object whose properties are date.Date parseable by [date] package, and value are the security price as a number.
func (s *Securities) Export(w io.Writer) error {

	type jsecurity struct {
		ID      string             `json:"id"`
		History map[string]float64 `json:"history"`
	}
	content := make(map[string]jsecurity)

	// Append securities for each ticker
	for ticker, sec := range s.content {
		// Create the json object security.
		js := jsecurity{
			ID:      string(sec.id),
			History: make(map[string]float64),
		}

		// fill the json security from security
		for day, value := range sec.prices.Values() {
			// error has been checked before
			js.History[day.String()] = value
		}
		content[ticker] = js
	}

	if err := json.NewEncoder(w).Encode(content); err != nil {
		return fmt.Errorf("cannot write Security format: %v", err)
	}
	return nil
}
