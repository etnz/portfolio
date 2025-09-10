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
	journal           *Journal
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
	balance, err := as.Balance(tx.When())
	if err != nil {
		return nil, fmt.Errorf("could not compute balance for validation: %w", err)
	}
	switch v := tx.(type) {
	case Buy:
		err = v.Validate(as.Ledger)
		return v, err
	case Sell:
		err = v.Validate(as.Ledger, balance)
		return v, err
	case Dividend:
		err = v.Validate(as.Ledger)
		return v, err
	case Deposit:
		err = v.Validate(as.Ledger)
		return v, err
	case Withdraw:
		err = v.Validate(as.Ledger)
		return v, err
	case Convert:
		err = v.Validate(as.Ledger)
		return v, err
	case Declare:
		err = v.Validate(as.Ledger)
		return v, err
	case Accrue:
		err = v.Validate(as.Ledger)
		return v, err
	default:
		return tx, fmt.Errorf("unsupported transaction type for validation: %T", tx)
	}
}
func (as *AccountingSystem) getJournal() (*Journal, error) {
	var err error
	if as.journal == nil {
		as.journal, err = as.newJournal()
	}
	if err != nil {
		return nil, err
	}
	return as.journal, nil
}

// Balance computes the Balance on a given day.
func (as *AccountingSystem) Balance(on Date) (*Balance, error) {
	j, err := as.getJournal()
	if err != nil {
		return nil, fmt.Errorf("could not get journal: %w", err)
	}
	balance, err := NewBalance(j, on, FIFO) // TODO: Make cost basis method configurable
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	return balance, nil
}

// DeclareSecurities scans all securities (and currencies) in the ledger and
// ensures they are declared in the marketdata. This function is crucial for
// maintaining consistency between the ledger's transactional records and the
// market data's security definitions.
func (as *AccountingSystem) DeclareSecurities() error {
	if err := ValidateCurrency(as.ReportingCurrency); err != nil {
		return fmt.Errorf("invalid default currency: %w", err)
	}

	for sec := range as.Ledger.AllSecurities() {
		as.MarketData.Add(sec)
	}
	for currency := range as.Ledger.AllCurrencies() {
		if currency == as.ReportingCurrency {
			// skip absurd self currency
			continue
		}
		id, err := NewCurrencyPair(currency, as.ReportingCurrency)
		if err != nil {
			return fmt.Errorf("could not create currency pair: %w", err)
		}
		as.MarketData.Add(NewSecurity(id, id.String(), currency))
	}
	return nil
}
