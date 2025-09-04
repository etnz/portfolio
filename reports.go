package portfolio

import (
	"github.com/etnz/portfolio/date"
)

// Performance holds the starting value and the calculated return for a specific period.
type Performance struct {
	StartValue float64
	Return     float64 // Return is a ratio (e.g., 0.05 for 5%)
}

// GainsReport contains the results of a capital gains calculation.
type GainsReport struct {
	Range             date.Range
	Method            CostBasisMethod
	ReportingCurrency string
	Securities        []SecurityGains
}

// SecurityGains holds the realized and unrealized gains for a single security.
type SecurityGains struct {
	Security    string
	Realized    float64
	Unrealized  float64
	Total       float64
	CostBasis   float64
	MarketValue float64
	Quantity    float64
}

// Summary provides a comprehensive, at-a-glance overview of the portfolio's
// state and performance on a given date.
type Summary struct {
	Date              date.Date
	ReportingCurrency string
	TotalMarketValue  float64
	Daily             Performance
	WTD               Performance // Week-to-Date
	MTD               Performance // Month-to-Date
	QTD               Performance // Quarter-to-Date
	YTD               Performance // Year-to-Date
	Inception         Performance
}
