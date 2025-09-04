package portfolio

import (
	"time"

	"github.com/etnz/portfolio/date"
)

// DailyReport provides a summary of a single day's portfolio changes, including
// a per-asset breakdown of performance.
type DailyReport struct {
	Date              date.Date
	Time              time.Time
	ReportingCurrency string
	ValueAtPrevClose  Money
	ValueAtClose      Money
	TotalGain         Money
	MarketGains       Money
	RealizedGains     Money
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
	return Percent(100 * r.TotalGain.AsMajorUnits() / r.ValueAtPrevClose.AsMajorUnits())
}

// HasBreakdown returns true if there is a breakdown of the day's gain.
func (r *DailyReport) HasBreakdown() bool {
	return !r.MarketGains.IsZero() || !r.RealizedGains.IsZero() || !r.NetCashFlow.IsZero()
}
