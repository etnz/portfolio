package portfolio

import (
	"github.com/etnz/portfolio/date"
)

// MarketData holds market data for a set of securities.
type MarketData struct {
	securities []*Security
	index      map[string]*Security
}

// NewMarketData returns a new empty market data.
func NewMarketData() *MarketData {
	return &MarketData{
		securities: make([]*Security, 0),
		index:      make(map[string]*Security),
	}
}

func (m *MarketData) Has(ticker string) bool {
	_, ok := m.index[ticker]
	return ok
}

func (m *MarketData) Get(ticker string) *Security { return m.index[ticker] }

// read a single value from the database for a given (ticker, day).
func (m *MarketData) read(ticker string, day date.Date) (float64, bool) {
	sec, ok := m.index[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.Get(day)
}
