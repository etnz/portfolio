package portfolio

import (
	"github.com/etnz/portfolio/date"
)

// Market holds market data for a set of securities.
type Market struct {
	securities []*Security
	index      map[string]*Security
}

// NewMarket returns a new empty market data collection.
func NewMarket() *Market {
	return &Market{
		securities: make([]*Security, 0),
		index:      make(map[string]*Security),
	}
}

func (m *Market) Has(ticker string) bool {
	_, ok := m.index[ticker]
	return ok
}

func (m *Market) Get(ticker string) *Security { return m.index[ticker] }

// read a single value from the database for a given (ticker, day).
func (m *Market) read(ticker string, day date.Date) (float64, bool) {
	sec, ok := m.index[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.Get(day)
}
