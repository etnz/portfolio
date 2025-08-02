package portfolio

import (
	"github.com/etnz/portfolio/date"
)

// Securities holds securities.
type Securities struct {
	securities []*Security
	index      map[string]*Security
}

// NewSecurities returns a new empty security database.
func NewSecurities() *Securities {
	return &Securities{
		securities: make([]*Security, 0),
		index:      make(map[string]*Security),
	}
}

func (s *Securities) Has(ticker string) bool {
	_, ok := s.index[ticker]
	return ok
}

func (s *Securities) Get(ticker string) *Security { return s.index[ticker] }

// read a single value from the database for a given (ticker, day).
func (s *Securities) read(ticker string, day date.Date) (float64, bool) {
	sec, ok := s.index[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.Get(day)
}
