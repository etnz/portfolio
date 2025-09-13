package portfolio

import (
	"time"
)

// HoldingReport represents a detailed view of portfolio holdings at a specific date.
type HoldingReport struct {
	Date              Date
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
	return c.Currency == other.Currency && c.Balance.Equal(other.Balance) && c.Value.Equal(other.Value)
}

// CounterpartyHolding represents the balance of a single counterparty account.
type CounterpartyHolding struct {
	Name     string
	Currency string
	Balance  Money
	Value    Money // In reporting currency
}

// NewHoldingReport calculates and returns a detailed holdings report for a given date.
func NewHoldingReport(ledger *Ledger, on Date, reportingCurrency string) (*HoldingReport, error) {
	report := &HoldingReport{
		Date:              on,
		Time:              time.Now(), // Generation time
		ReportingCurrency: reportingCurrency,
		Securities:        []SecurityHolding{},
		Cash:              []CashHolding{},
		Counterparties:    []CounterpartyHolding{},
	}

	journal, err := newJournal(ledger, reportingCurrency)
	if err != nil {
		return nil, err
	}
	balance, err := NewBalance(journal, on, FIFO)
	if err != nil {
		return nil, err
	}

	// Securities
	for sec := range balance.Securities() {
		ticker := sec.Ticker()
		id := sec.ID()
		currency := sec.Currency()
		position := balance.Position(ticker)
		if position.IsZero() {
			continue
		}
		report.Securities = append(report.Securities, SecurityHolding{
			Ticker:      ticker,
			ID:          id.String(),
			Currency:    currency,
			Quantity:    position,
			Price:       balance.Price(ticker),
			MarketValue: balance.Convert(balance.MarketValue(ticker)),
		})
	}

	// Cash
	for currency := range balance.Currencies() {
		bal := balance.Cash(currency)
		if bal.IsZero() {
			continue
		}
		convertedBalance := balance.Convert(bal)
		report.Cash = append(report.Cash, CashHolding{
			Currency: currency,
			Balance:  bal,
			Value:    convertedBalance,
		})
	}

	// Counterparties
	for account := range balance.Counterparties() {
		bal, currency := balance.Counterparty(account), balance.CounterpartyCurrency(account)
		if bal.IsZero() {
			continue
		}
		convertedBalance := balance.Convert(bal)
		report.Counterparties = append(report.Counterparties, CounterpartyHolding{
			Name:     account,
			Currency: currency,
			Balance:  bal,
			Value:    convertedBalance,
		})
	}

	report.TotalValue = balance.TotalPortfolioValue()

	return report, nil
}
