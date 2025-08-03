package portfolio

// Validate checks a transaction for correctness and applies quick fixes where
// applicable (e.g., resolving "sell all"). It returns the validated (and
// potentially modified) transaction or an error detailing any validation failures.
func Validate(market *MarketData, ledger *Ledger, tx Transaction) (ntx Transaction, err error) {
	switch v := tx.(type) {
	case Buy:
		err = v.Validate(market, ledger)
	case Sell:
		err = v.Validate(market, ledger)
	case Dividend:
		err = v.Validate(market, ledger)
	case Deposit:
		err = v.Validate(market, ledger)
	case Withdraw:
		err = v.Validate(market, ledger)
	case Convert:
		err = v.Validate(market, ledger)
	}
	return tx, err
}
