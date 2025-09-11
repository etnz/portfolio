package portfolio

import (
	"fmt"
)

// AccountingSystem encapsulates all the data required for portfolio management,
// combining transactional data with market data. It serves as a central point
// of access for querying portfolio state, such as positions and cash balances,
// and for validating new transactions.
//
// By holding both the Ledger (the record of all transactions) and the MarketData
// (the repository of security information and prices), it provides the complete
// context needed for most portfolio operations.
type AccountingSystem struct {
	Ledger            *Ledger
	MarketData        *MarketData
	ReportingCurrency string
}

// NewAccountingSystem creates a new accounting system from a ledger and market data.
func NewAccountingSystem(ledger *Ledger, marketData *MarketData, reportingCurrency string) (*AccountingSystem, error) {
	if reportingCurrency != "" {
		if err := ValidateCurrency(reportingCurrency); err != nil {
			return nil, fmt.Errorf("invalid reporting currency: %w", err)
		}
	}
	as := &AccountingSystem{
		Ledger:            ledger,
		MarketData:        marketData,
		ReportingCurrency: reportingCurrency,
	}
	// if err := as.declareSecurities(); err != nil {
	// 	return nil, fmt.Errorf("could not declare securities from ledger to the market data: %w", err)
	// }

	return as, nil
}

// Validate checks a transaction for correctness and applies quick fixes where
// applicable (e.g., resolving "sell all"). It returns the validated (and
// potentially modified) transaction or an error detailing any validation failures.
func (as *AccountingSystem) Validate(tx Transaction) (Transaction, error) {
	// For validations that need the state of the portfolio, we compute the balance
	// on the transaction date.
	var err error
	switch v := tx.(type) {
	case Buy:
		err = v.Validate(as.Ledger)
	case Sell:
		// todo: recomputing the whole journal everytime is expensive.
		var j *Journal
		j, err = NewJournal(as.Ledger, as.MarketData, as.ReportingCurrency)
		if err != nil {
			return nil, fmt.Errorf("Invalid journal: %w", err)
		}

		balance, e := NewBalance(j, tx.When(), FIFO) // TODO: Make cost basis method configurable
		if e != nil {
			return nil, fmt.Errorf("could not create balance from journal: %w", e)
		}
		err = v.Validate(as.Ledger, balance)
	case Dividend:
		err = v.Validate(as.Ledger)
	case Deposit:
		err = v.Validate(as.Ledger)
	case Withdraw:
		err = v.Validate(as.Ledger)
	case Convert:
		err = v.Validate(as.Ledger)
	case Declare:
		err = v.Validate(as.Ledger)
	case Accrue:
		err = v.Validate(as.Ledger)
	default:
		return tx, fmt.Errorf("unsupported transaction type for validation: %T %v", tx, tx)
	}
	if err != nil {
		return tx, fmt.Errorf("invalid %s transaction on %v: %w", tx.What(), tx.When(), err)
	}
	return tx, nil
}
func (as *AccountingSystem) newJournal() (*Journal, error) {
	return NewJournal(as.Ledger, as.MarketData, as.ReportingCurrency)
}

// Balance computes the Balance on a given day.
func (as *AccountingSystem) Balance(on Date) (*Balance, error) {
	j, err := as.newJournal()
	if err != nil {
		return nil, fmt.Errorf("could not get journal: %w", err)
	}
	balance, err := NewBalance(j, on, FIFO) // TODO: Make cost basis method configurable
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	return balance, nil
}

// DeclareSecurities scans all securities and currencies in the ledger and
// ensures they are declared in the market data. This function is crucial for
// maintaining consistency between the ledger's transactional records and the market
// data's security definitions.
func DeclareSecurities(ledger *Ledger, marketData *MarketData, reportingCurrency string) error {
	if err := ValidateCurrency(reportingCurrency); err != nil {
		return fmt.Errorf("invalid default currency: %w", err)
	}

	for sec := range ledger.AllSecurities() {
		marketData.Add(sec)
	}
	for currency := range ledger.AllCurrencies() {
		if currency == reportingCurrency {
			// skip absurd self currency
			continue
		}
		id, err := NewCurrencyPair(currency, reportingCurrency)
		if err != nil {
			return fmt.Errorf("could not create currency pair: %w", err)
		}
		marketData.Add(NewSecurity(id, id.String(), currency))
	}
	return nil
}
