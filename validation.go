package portfolio

// Validate 'tx' and return a copy with quick fixes apply or an error with all validation failures.
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
