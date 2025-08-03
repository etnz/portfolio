package portfolio

import "fmt"

// Validation layer is done in multiple steps:
// 1. transactions are validated internally
// 2. transactions are validated against the known market data for consistency
// 3/ transactions are validated against the history of transactions.

// Validate checks a ledger of transactions for internal and contextual consistency.
// It performs two main checks:
//  1. Calls the Validate() method on each transaction for self-validation.
//  2. For transactions involving a security (Buy, Sell, Dividend), it ensures
//     the security exists in the provided Market data.
func Validate(market *MarketData, ledger *Ledger, tx Transaction) error {
	// 1. Internal validation of the transaction itself.
	if err := tx.Validate(); err != nil {
		return fmt.Errorf("transaction (%s on %s) is invalid: %w", tx.What(), tx.When(), err)
	}

	// 2. Contextual validation against the securities database.
	var securityTicker string
	switch v := tx.(type) {
	case Buy:
		securityTicker = v.Security
	case Sell:
		securityTicker = v.Security
	case Dividend:
		securityTicker = v.Security
	}

	if securityTicker != "" {
		if !market.Has(securityTicker) {
			return fmt.Errorf("transaction (%s on %s) references non-existent security ticker %q", tx.What(), tx.When(), securityTicker)
		}
	}
	return nil
}

// Step 3 (validating against transaction history) would likely be part of a
// Portfolio object's method that processes transactions sequentially and
// maintains the state of holdings, like ensuring you don't sell more shares
// than you own. This function provides the initial, static validation.
