package portfolio

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/etnz/portfolio/date"
)

// This file contains functions to handle the import/export format.
// The format is designed to be human-readable, single-file, and easy to merge into a database.

// ImportMarketData imports market data from 'r' in the import/export format.
//
// The import format is a JSONL file, where each line is a JSON object representing a security.
// For example:
//
//	{"ticker":"AAPL","id":"US0378331005.XNAS","currency":"USD","history":{"2023-01-02":130.28,"2023-01-03":125.07}}
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
		Splits   []Split            `json:"splits,omitempty"`
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
		sec := Security{
			ticker:   js.Ticker,
			id:       ID(js.ID),
			currency: js.Currency,
		}
		m.Add(sec)

		// fill the security from json
		for day, value := range js.History {
			// error has been checked before
			d, _ := date.Parse(day)
			m.Append(sec.ID(), d, value)
		}
		for _, split := range js.Splits {
			m.AddSplit(sec.ID(), split)
		}
	}
	return m, nil
}

// ExportMarketData exports the market data to 'w' in the import/export format.
//
// The format is a JSONL file, where each line is a JSON object representing a security.
// For example:
//
//	{"ticker":"AAPL","id":"US0378331005.XNAS","currency":"USD","history":{"2023-01-02":130.28,"2023-01-03":125.07}}
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
		Splits   []Split            `json:"splits,omitempty"`
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
		// Create the json object security.
		js := jsecurity{
			Ticker:   sec.Ticker(),
			ID:       sec.ID().String(),
			Currency: sec.currency,
			History:  make(map[string]float64),
			Splits:   m.Splits(sec.ID()),
		}

		for day, value := range m.Prices(sec.ID()) {
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
