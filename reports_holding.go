package portfolio

import (
	"time"

	"github.com/etnz/portfolio/date"
)

// HoldingReport represents a detailed view of portfolio holdings at a specific date.
type HoldingReport struct {
	Date              date.Date
	Time              time.Time // Generation time
	ReportingCurrency string
	Securities        []SecurityHolding
	Cash              []CashHolding
	Counterparties    []CounterpartyHolding
	TotalValue        Money
}

// SecurityHolding represents the holding of a single security.
type SecurityHolding struct {
	Ticker      string
	ID          string
	Currency    string
	Quantity    Quantity
	Price       Money
	MarketValue Money // In reporting currency
}

// CashHolding represents the balance of a single currency.
type CashHolding struct {
	Currency string
	Balance  Money // in self money
	Value    Money // In reporting currency
}

func (c CashHolding) Equals(other CashHolding) bool {
	return c.Currency == other.Currency && c.Balance.Equals(other.Balance) && c.Value.Equals(other.Value)
}

// CounterpartyHolding represents the balance of a single counterparty account.
type CounterpartyHolding struct {
	Name     string
	Currency string
	Balance  Money
	Value    Money // In reporting currency
}
