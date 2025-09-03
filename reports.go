package portfolio

import (
	"time"

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

// DailyReport provides a summary of a single day's portfolio changes, including
// a per-asset breakdown of performance.
type DailyReport struct {
	Date              date.Date
	Time              time.Time
	ReportingCurrency string
	ValueAtPrevClose  float64
	ValueAtClose      float64
	TotalGain         float64
	MarketGains       float64
	RealizedGains     float64
	NetCashFlow       float64
	ActiveAssets      []AssetGain
	Transactions      []Transaction
}

// AssetGain represents the daily gain or loss for a single security.
type AssetGain struct {
	Security string
	Gain     float64
	Return   float64
}

// HoldingReport represents a detailed view of portfolio holdings at a specific date.
type HoldingReport struct {
	Date              date.Date
	ReportingCurrency string
	Securities        []SecurityHolding
	Cash              []CashHolding
	Counterparties    []CounterpartyHolding
	TotalValue        float64
}

// SecurityHolding represents the holding of a single security.
type SecurityHolding struct {
	Ticker      string
	ID          string
	Currency    string
	Quantity    float64
	Price       float64
	MarketValue float64 // In reporting currency
}

// CashHolding represents the balance of a single currency.
type CashHolding struct {
	Currency string
	Balance  float64
	Value    float64 // In reporting currency
}

// CounterpartyHolding represents the balance of a single counterparty account.
type CounterpartyHolding struct {
	Name     string
	Currency string
	Balance  float64
	Value    float64 // In reporting currency
}
