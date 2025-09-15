package portfolio

import (
	"time"
)

// DailyReport provides a summary of a single day's portfolio changes, including
// a per-asset breakdown of performance.
type DailyReport struct {
	Date              Date
	Time              time.Time
	ReportingCurrency string
	ValueAtPrevClose  Money
	ValueAtClose      Money
	TotalGain         Money
	MarketGains       Money
	RealizedGains     Money
	Dividends         Money
	NetCashFlow       Money
	ActiveAssets      []AssetGain
	Transactions      []Transaction
}

// AssetGain represents the daily gain or loss for a single security.
type AssetGain struct {
	Security string
	Gain     Money
	Return   Percent
}

// PercentageGain returns the percentage gain for the day.
func (r *DailyReport) PercentageGain() Percent {
	if r.ValueAtPrevClose.IsZero() {
		return 0
	}
	return Percent(100 * r.TotalGain.AsFloat() / r.ValueAtPrevClose.AsFloat())
}

// HasBreakdown returns true if there is a breakdown of the day's gain.
func (r *DailyReport) HasBreakdown() bool {
	return !r.MarketGains.IsZero() || !r.RealizedGains.IsZero() || !r.NetCashFlow.IsZero()
}

// NewDailyReport calculates and returns a summary of the portfolio's performance for a single day from a given ledger.
func NewDailyReport(ledger *Ledger, on Date, reportingCurrency string) (*DailyReport, error) {
	journal, err := newJournal(ledger, reportingCurrency)
	if err != nil {
		return nil, err
	}

	endBalance, err := NewBalance(journal, on, AverageCost)
	if err != nil {
		return nil, err
	}
	// 1. Calculate value at previous day's close
	startBalance, err := NewBalance(journal, on.Add(-1), AverageCost)
	if err != nil {
		return nil, err
	}

	report := &DailyReport{
		Date:              on,
		Time:              time.Now(), // Generation time
		ReportingCurrency: reportingCurrency,
		ActiveAssets:      []AssetGain{},
		Transactions:      []Transaction{},
	}

	report.ValueAtClose = endBalance.TotalPortfolioValue()
	report.ValueAtPrevClose = startBalance.TotalPortfolioValue()

	// 3. Get all transactions for the specified day
	for _, tx := range ledger.transactions {
		if tx.When() == on {
			report.Transactions = append(report.Transactions, tx)
		}
	}

	// 4. Calculate Net Cash Flow and Realized Gains for the day
	report.NetCashFlow = endBalance.TotalCashFlow().Sub(startBalance.TotalCashFlow())
	report.Dividends = endBalance.TotalDividendsReceived().Sub(startBalance.TotalDividendsReceived())

	// Gains have to be computed per security (in fact per securities currency)
	// then converted to the reporting currency.
	var totalRealized Money
	for sec := range endBalance.Securities() {
		ticker := sec.Ticker()
		gain := endBalance.RealizedGain(ticker).Sub(startBalance.RealizedGain(ticker))
		gain = endBalance.Convert(gain)
		totalRealized = totalRealized.Add(gain)
	}
	report.RealizedGains = totalRealized

	// 5. Calculate Total Gain and Market Gains
	report.TotalGain = report.ValueAtClose.Sub(report.ValueAtPrevClose)
	// MarketGains is the change in value not accounted for by cash flows, realized gains, or dividends.
	report.MarketGains = report.TotalGain.Sub(report.NetCashFlow).Sub(report.RealizedGains).Sub(report.Dividends)

	// 6. Calculate Active Asset Gains
	for sec := range endBalance.Securities() {

		if endBalance.Position(sec.Ticker()).IsZero() {
			continue // ignore assets not held today
		}

		valueToday := endBalance.MarketValue(sec.Ticker())
		valuePrev := startBalance.MarketValue(sec.Ticker())
		gain := valueToday.Sub(valuePrev)

		// Adjust for buys/sells during the day
		for _, tx := range report.Transactions {
			switch v := tx.(type) {
			case Buy:
				if v.Security == sec.Ticker() {
					cost := v.Amount
					gain = gain.Sub(cost)
				}
			case Sell:
				if v.Security == sec.Ticker() {
					proceeds := v.Amount
					gain = gain.Add(proceeds)
				}
			}
		}

		yield := 0.0
		if !valuePrev.IsZero() { // if there was an initial value
			yield = gain.AsFloat() / valuePrev.AsFloat() * 100.0
		}

		// convert to reporting currency
		gain = endBalance.Convert(gain)
		assetGain := AssetGain{
			Security: sec.Ticker(),
			Gain:     gain,
			Return:   Percent(yield),
		}

		report.ActiveAssets = append(report.ActiveAssets, assetGain)

	}

	return report, nil
}
