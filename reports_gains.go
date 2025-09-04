package portfolio

import "github.com/etnz/portfolio/date"

// GainsReport contains the results of a capital gains calculation.
type GainsReport struct {
	Range             date.Range
	Method            CostBasisMethod
	ReportingCurrency string
	Securities        []SecurityGains
	Realized          Money
	Unrealized        Money
	Total             Money
}

// SecurityGains holds the realized and unrealized gains for a single security.
type SecurityGains struct {
	Security    string
	Realized    Money
	Unrealized  Money
	Total       Money
	CostBasis   Money
	MarketValue Money
	Quantity    Quantity
}
