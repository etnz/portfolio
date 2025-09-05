package portfolio

import (
	"github.com/etnz/portfolio/date"
)

// Performance holds the starting value and the calculated return for a specific period.
type Performance struct {
	StartValue Money
	Return     Percent // Return is a ratio (e.g., 0.05 for 5%)
}

// Summary provides a comprehensive, at-a-glance overview of the portfolio's
// state and performance on a given date.
type Summary struct {
	Date              date.Date
	ReportingCurrency string
	TotalMarketValue  Money
	Daily             Performance
	WTD               Performance // Week-to-Date
	MTD               Performance // Month-to-Date
	QTD               Performance // Quarter-to-Date
	YTD               Performance // Year-to-Date
	Inception         Performance
}
