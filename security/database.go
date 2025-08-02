package security

import (
	"github.com/etnz/portfolio/date"
)

// Securities holds securities.
type Securities struct {
	content map[string]*Security
}

// New returns a new empty database.
func New() *Securities { return &Securities{make(map[string]*Security)} }

func (s *Securities) Has(ticker string) bool {
	_, ok := s.content[ticker]
	return ok
}

func (s *Securities) Get(ticker string) *Security { return s.content[ticker] }

// read a single value from the database for a given (ticker, day).
func (s *Securities) read(ticker string, day date.Date) (float64, bool) {
	sec, ok := s.content[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.Get(day)
}
