package portfolio

import "fmt"

// AccountingSystem encapsulates all the data required for portfolio management,
// combining transactional data with market data. It serves as a central point
// of access for querying portfolio state, such as positions and cash balances,
// and for validating new transactions.
//
// By holding both the Ledger (the record of all transactions) and the MarketData
// (the repository of security information and prices), it provides the complete
// context needed for most portfolio operations.
type AccountingSystem struct {
	Ledger     *Ledger
	MarketData *MarketData
}

// NewAccountingSystem creates a new accounting system from a ledger and market data.
func NewAccountingSystem(ledger *Ledger, marketData *MarketData) *AccountingSystem {
	return &AccountingSystem{
		Ledger:     ledger,
		MarketData: marketData,
	}
}

// Validate checks a transaction for correctness and applies quick fixes where
// applicable (e.g., resolving "sell all"). It returns the validated (and
// potentially modified) transaction or an error detailing any validation failures.
func (as *AccountingSystem) Validate(tx Transaction) (Transaction, error) {
	var err error
	// The type switch creates a copy (v) of the transaction struct.
	// We must call Validate on a pointer to this copy (&v) to allow modifications.
	// We then return the (potentially modified) copy.
	switch v := tx.(type) {
	case Buy:
		err = (&v).Validate(as)
		return v, err
	case Sell:
		err = (&v).Validate(as)
		return v, err
	case Dividend:
		err = (&v).Validate(as)
		return v, err
	case Deposit:
		err = (&v).Validate(as)
		return v, err
	case Withdraw:
		err = (&v).Validate(as)
		return v, err
	case Convert:
		err = (&v).Validate(as)
		return v, err
	default:
		return tx, fmt.Errorf("unsupported transaction type for validation: %T", tx)
	}
}
