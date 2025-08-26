package portfolio

import "github.com/etnz/portfolio/date"

// HoldingReport represents a detailed view of portfolio holdings at a specific date.
type HoldingReport struct {
	Date              date.Date
	ReportingCurrency string
	Securities        []SecurityHolding
	Cash              []CashHolding
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
	Currency         string
	Balance          float64
	Value            float64 // In reporting currency
}
