package portfolio

import "fmt"

// Summary provides a comprehensive, at-a-glance overview of the portfolio's
// state and performance on a given date.
type Summary struct {
	Date              Date
	ReportingCurrency string
	TotalMarketValue  Money
	Daily             Performance
	WTD               Performance // Week-to-Date
	MTD               Performance // Month-to-Date
	QTD               Performance // Quarter-to-Date
	YTD               Performance // Year-to-Date
	Inception         Performance
}

// NewSummary calculates and returns a comprehensive summary of the portfolio's
// state and performance on a given date.
func NewSummary(ledger *Ledger, on Date, reportingCurrency string) (*Summary, error) {
	if reportingCurrency == "" {
		return nil, fmt.Errorf("reporting currency is not set in accounting system")
	}

	summary := &Summary{
		Date:              on,
		ReportingCurrency: reportingCurrency,
	}

	journal, err := newJournal(ledger, reportingCurrency)
	if err != nil {
		return nil, err
	}

	endBalance, err := NewBalance(journal, on, AverageCost)
	if err != nil {
		return nil, err
	}

	yesterdayBalance, err := NewBalance(journal, on.Add(-1), AverageCost)
	if err != nil {
		return nil, err
	}

	weekBalance, err := NewBalance(journal, on.StartOf(Weekly).Add(-1), AverageCost)
	if err != nil {
		return nil, err
	}

	monthBalance, err := NewBalance(journal, on.StartOf(Monthly).Add(-1), AverageCost)
	if err != nil {
		return nil, err
	}

	quarterBalance, err := NewBalance(journal, on.StartOf(Quarterly).Add(-1), AverageCost)
	if err != nil {
		return nil, err
	}

	yearBalance, err := NewBalance(journal, on.StartOf(Yearly).Add(-1), AverageCost)
	if err != nil {
		return nil, err
	}

	// 1. Calculate current total market value
	summary.TotalMarketValue = endBalance.TotalPortfolioValue()

	// 2. Calculate performance for each period
	periodTWR := func(start *Balance) (perf Performance) {
		perf = NewPerformance(start.TotalPortfolioValue(), endBalance.TotalPortfolioValue())
		perf.Return = Percent(endBalance.linkedTWR/start.linkedTWR - 1)
		return perf
	}

	summary.Daily = periodTWR(yesterdayBalance)
	summary.WTD = periodTWR(weekBalance)
	summary.MTD = periodTWR(monthBalance)
	summary.QTD = periodTWR(quarterBalance)
	summary.YTD = periodTWR(yearBalance)
	summary.Inception = Performance{
		Return: Percent(endBalance.linkedTWR - 1),
	}
	return summary, nil
}
