package portfolio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/etnz/portfolio/date"
)

// This file contains functions to handle the import/export format.
// The format is designed to be human-readable, single-file, and easy to merge into a database.

// ImportMarketData imports market data from 'r' in the import/export format.
//
// The import format is a JSONL file, where each line is a JSON object representing a security.
//
// A security is a single json object whose property 'ticker' contains the security ticker, 'id' contains the security ID as string, and property 'history' contains a single json object representing the security history.
//
// The security history is represented as a single json object whose properties are date.Date parseable by [date] package, and value are the security price as a number.
func ImportMarketData(r io.Reader) (*MarketData, error) {

	// the readable version of the format is can be summarized by a few types.
	type jsecurity struct {
		Ticker   string             `json:"ticker"`
		ID       string             `json:"id"`
		Currency string             `json:"currency"`
		History  map[string]float64 `json:"history"`
	}

	var jsecurities []jsecurity
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(strings.TrimSpace(string(line))) == 0 {
			continue
		}
		var js jsecurity
		if err := json.Unmarshal(line, &js); err != nil {
			return nil, fmt.Errorf("cannot parse line for Security import format: %q: %w", string(line), err)
		}
		jsecurities = append(jsecurities, js)
	}

	m := NewMarketData()

	// Append securities for each ticker
	for _, js := range jsecurities {
		// Create the security.
		sec := &Security{
			ticker:   js.Ticker,
			id:       ID(js.ID),
			currency: js.Currency,
		}

		// fill the security from json
		for day, value := range js.History {
			// error has been checked before
			d, _ := date.Parse(day)
			sec.prices.Append(d, value)
		}
		m.securities = append(m.securities, sec)
		m.index[sec.ticker] = sec
	}
	slices.SortFunc(m.securities, func(a, b *Security) int {
		return strings.Compare(a.ticker, b.ticker)
	})
	return m, nil
}

// ExportMarketData exports the market data to 'w' in the import/export format.
//
// The format is a JSONL file, where each line is a JSON object representing a security.
//
// A security is a single json object whose property 'ticker' contains the security ticker, 'id' contains the security ID as string, and property 'history' contains a single json object representing the security history.
//
// The security history is represented as a single json object whose properties are date.Date parseable by [date] package, and value are the security price as a number.
func ExportMarketData(w io.Writer, m *MarketData) error {

	type jsecurity struct {
		Ticker   string             `json:"ticker"`
		ID       string             `json:"id"`
		Currency string             `json:"currency,omitempty"`
		History  map[string]float64 `json:"history"`
	}

	for _, sec := range m.securities {
		// Create the json object security.
		js := jsecurity{
			Ticker:   sec.Ticker(),
			ID:       string(sec.id),
			Currency: sec.currency,
			History:  make(map[string]float64),
		}

		for day, value := range sec.prices.Values() {
			js.History[day.String()] = value
		}

		data, err := json.Marshal(js)
		if err != nil {
			return fmt.Errorf("cannot marshal security %q: %w", sec.Ticker(), err)
		}
		if _, err := w.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("cannot write Security format: %w", err)
		}
	}
	return nil
}
