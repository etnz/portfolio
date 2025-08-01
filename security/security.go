package security

import "github.com/etnz/porfolio/date"

// ID represent the unique identifier of a security. It must follow a specific format.
type ID string

// Security represent a publicly or privately tradeable asset, stock, ETF, house.
type Security struct {
	id     ID
	ticker string                 // The ticker used in portfolio and human friendly persistence format.
	prices *date.History[float64] // the price historical value.
}

func (s *Security) ID() ID {
	return s.id
}

func (s *Security) Ticker() string {
	return s.ticker
}

func (s *Security) Prices() *date.History[float64] {
	return s.prices
}
