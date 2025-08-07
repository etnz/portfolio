package portfolio

import (
	"github.com/etnz/portfolio/date"
)

// MarketData holds market data for a set of securities.
type MarketData struct {
	securities  []*Security
	indexTicker map[string]*Security
	indexID     map[ID]*Security
}

// Resolve the market data ticker to its ID.
//
// If missing return the zero value.
func (m *MarketData) Resolve(ticker string) ID {
	sec, ok := m.indexTicker[ticker]
	if !ok || sec == nil {
		return ""
	}
	return sec.ID()
}

func (m *MarketData) Add(sec *Security) {
	m.securities = append(m.securities, sec)
	m.indexTicker[sec.ticker] = sec
	m.indexID[sec.ID()] = sec
}

// NewMarketData returns a new empty market data.
func NewMarketData() *MarketData {
	return &MarketData{
		securities:  make([]*Security, 0),
		indexTicker: make(map[string]*Security),
		indexID:     make(map[ID]*Security),
	}
}

// Has checks if a security with the given ticker exists in the market data.
func (m *MarketData) Has(ticker string) bool {
	_, ok := m.indexTicker[ticker]
	return ok
}

// Get retrieves a security by its ticker. It returns nil if the security is not found.
func (m *MarketData) Get(id ID) *Security { return m.indexID[id] }

// read retrieves the price for a given security ticker on a specific day.
// It returns the price and true if found, otherwise it returns 0.0 and false.
func (m *MarketData) read(ticker string, day date.Date) (float64, bool) {
	sec, ok := m.indexTicker[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.Get(day)
}

// PriceAsOf returns the price of a security on a given day, or the most recent price before it.
func (m *MarketData) PriceAsOf(id ID, day date.Date) (float64, bool) {
	sec, ok := m.indexID[id]
	if !ok {
		return 0.0, false
	}
	return sec.prices.ValueAsOf(day)
}
