package portfolio

import (
	"iter"

	"github.com/etnz/portfolio/date"
)

// MarketData holds all the market data, including security definitions and their price histories.
type MarketData struct {
	securities map[ID]Security
	tickers    map[string]ID
	prices     map[ID]*date.History[float64]
}

// NewMarketData creates an empty MarketData store.
func NewMarketData() *MarketData {
	return &MarketData{
		securities: make(map[ID]Security),
		tickers:    make(map[string]ID),
		prices:     make(map[ID]*date.History[float64]),
	}
}

// Add adds a security to the market data. It also initializes an empty price history for it.
func (m *MarketData) Add(s Security) {
	if _, ok := m.securities[s.ID()]; ok {
		return
	}
	m.securities[s.ID()] = s
	m.tickers[s.Ticker()] = s.ID()
	m.prices[s.ID()] = &date.History[float64]{}
}

// Get retrieves a security by its ID. It returns zero if the security is not found.
func (m *MarketData) Get(id ID) Security { return m.securities[id] }

// Resolve converts a ticker to a security ID.
func (m *MarketData) Resolve(ticker string) ID {
	return m.tickers[ticker]
}

// PriceAsOf returns the price of a security on a given date.
func (m *MarketData) PriceAsOf(id ID, on date.Date) (float64, bool) {
	if prices, ok := m.prices[id]; ok {
		return prices.ValueAsOf(on)
	}
	return 0, false
}

func (m *MarketData) Append(id ID, day date.Date, price float64) bool {
	if prices, ok := m.prices[id]; ok {
		prices.Append(day, price)
		return true
	}
	return false
}

// Values return a iterator on date and prices for the given ID (or nil)
func (m *MarketData) Prices(id ID) iter.Seq2[date.Date, float64] {
	prices, ok := m.prices[id]
	if !ok {
		return func(yield func(date.Date, float64) bool) {}
	}
	return prices.Values()

}

// Has checks if a security with the given ticker exists in the market data.
func (m *MarketData) Has(ticker string) bool {
	_, ok := m.tickers[ticker]
	return ok
}

// read retrieves the price for a given security on a specific day.
// It returns the price and true if found, otherwise it returns 0.0 and false.
func (m *MarketData) read(id ID, day date.Date) (float64, bool) {
	prices, ok := m.prices[id]
	if !ok {
		return 0.0, false
	}
	return prices.Get(day)
}