package portfolio

import (
	"github.com/etnz/portfolio/date"
)

// MarketData holds market data for a set of securities.
type MarketData struct {
	securities []*Security
	index      map[string]*Security
}

func (m *MarketData) Add(sec *Security) {
	m.securities = append(m.securities, sec)
	m.index[sec.ticker] = sec
}

// NewMarketData returns a new empty market data.
func NewMarketData() *MarketData {
	return &MarketData{
		securities: make([]*Security, 0),
		index:      make(map[string]*Security),
	}
}

// Has checks if a security with the given ticker exists in the market data.
func (m *MarketData) Has(ticker string) bool {
	_, ok := m.index[ticker]
	return ok
}

// Get retrieves a security by its ticker. It returns nil if the security is not found.
func (m *MarketData) Get(ticker string) *Security { return m.index[ticker] }

// read retrieves the price for a given security ticker on a specific day.
// It returns the price and true if found, otherwise it returns 0.0 and false.
func (m *MarketData) read(ticker string, day date.Date) (float64, bool) {
	sec, ok := m.index[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.Get(day)
}

// PriceAsOf returns the price of a security on a given day, or the most recent price before it.
func (m *MarketData) PriceAsOf(ticker string, day date.Date) (float64, bool) {
	sec, ok := m.index[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.ValueAsOf(day)
}
