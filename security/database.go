package security

import (
	"github.com/etnz/porfolio/date"
)

// DB holds the all the securities data.
type DB struct {
	content map[string]*Security
}

// NewDB returns a new empty database.
func NewDB() *DB { return &DB{make(map[string]*Security)} }

func (db *DB) Has(ticker string) bool {
	_, ok := db.content[ticker]
	return ok
}

// read a single value from the database for a given (ticker, day).
func (db *DB) read(ticker string, day date.Date) (float64, bool) {
	sec, ok := db.content[ticker]
	if !ok {
		return 0.0, false
	}
	return sec.prices.Get(day)
}
