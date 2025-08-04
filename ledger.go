package portfolio

import (
	"iter"

	"sort"

	"github.com/etnz/portfolio/date"
)

// Ledger represents a list of transactions.
type Ledger struct {
	transactions []Transaction
}

// NewLedger creates an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{
		transactions: make([]Transaction, 0),
	}
}

// Transactions returns an iterator that yields each transaction in its original order.
func (l Ledger) Transactions() iter.Seq2[int, Transaction] {
	// The returned iterator preserves the original order of transactions in the ledger.
	return func(yield func(int, Transaction) bool) {
		for i, tx := range l.transactions {
			if !yield(i, tx) {
				return
			}
		}
	}
}

// stableSort sorts the ledger by transaction date. The sort is stable, meaning
// transactions on the same day maintain their original relative order.
func (l *Ledger) stableSort() {
	sort.SliceStable(l.transactions, func(i, j int) bool {
		return l.transactions[i].When().Before(l.transactions[j].When())
	})
}

// Position computes the total quantity of a security held on a specific date by
// summing up all buy and sell transactions for that security up to and including that date.
func (l *Ledger) Position(ticker string, on date.Date) float64 {
	var quantity float64
	for _, tx := range l.SecurityTransactions(ticker, on) {
		switch v := tx.(type) {
		case Buy:
			quantity += v.Quantity
		case Sell:
			// The quantity for a Sell transaction should have been resolved to a
			// concrete value during validation. A quantity of 0 in a Sell
			// transaction means "sell all" and is resolved before being stored.
			quantity -= v.Quantity
		}
	}
	return quantity
}

// CashBalance computes the total cash in a specific currency on a specific date.
func (l *Ledger) CashBalance(currency string, on date.Date) float64 {
	var balance float64
	for _, tx := range l.transactions {
		if tx.When().After(on) {
			// The ledger is sorted by date, so it's safe to break.
			break
		}
		switch v := tx.(type) {
		case Buy:
			if v.Currency == currency {
				balance -= v.Quantity * v.Price
			}
		case Sell:
			if v.Currency == currency {
				balance += v.Quantity * v.Price
			}
		case Dividend:
			if v.Currency == currency {
				balance += v.Amount
			}
		case Deposit:
			if v.Currency == currency {
				balance += v.Amount
			}
		case Withdraw:
			if v.Currency == currency {
				balance -= v.Amount
			}
		case Convert:
			if v.FromCurrency == currency {
				balance -= v.FromAmount
			}
			if v.ToCurrency == currency {
				balance += v.ToAmount
			}
		}
	}
	return balance
}

// SecurityTransactions returns an iterator over transactions related to a specific
// security (Buy, Sell, Dividend) up to and including a given date.
// The ledger must be sorted by date for this to work correctly.
func (l Ledger) SecurityTransactions(ticker string, max date.Date) iter.Seq2[int, Transaction] {
	return func(yield func(int, Transaction) bool) {
		for i, tx := range l.transactions {

			if tx.When().After(max) {
				// The ledger is sorted by dated, so it's safe to return.
				return
			}
			switch v := tx.(type) {
			case Buy:
				if v.Security == ticker {
					if !yield(i, tx) {
						return
					}
				}
			case Sell:
				if v.Security == ticker {
					if !yield(i, tx) {
						return
					}
				}
			case Dividend:
				if v.Security == ticker {
					if !yield(i, tx) {
						return
					}
				}
			}
		}
	}
}

// AllSecurities iterates over security tickers that appear
// in the ledger's transactions (Buy, Sell, or Dividend) without repetition.
func (l *Ledger) AllSecurities() iter.Seq[string] {
	return func(yield func(string) bool) {
		visitedTickers := make(map[string]struct{})
		for _, tx := range l.transactions {
			var security string
			switch v := tx.(type) {
			case Buy:
				security = v.Security
			case Sell:
				security = v.Security
			case Dividend:
				security = v.Security
			}
			if _, visited := visitedTickers[security]; !visited {
				visitedTickers[security] = struct{}{}
				if !yield(security) {
					return
				}
			}
		}
	}
}

// AllCurrencies iterates over all currencies that appear in the ledger.
// in the ledger's transactions.
func (l *Ledger) AllCurrencies() iter.Seq[string] {
	return func(yield func(string) bool) {
		visitedCurrencies := make(map[string]struct{})
		visit := func(currency string) bool {
			if _, visited := visitedCurrencies[currency]; !visited {
				visitedCurrencies[currency] = struct{}{}
				return yield(currency)
			}
			return false
		}
		for _, tx := range l.transactions {
			switch v := tx.(type) {
			case Buy:
				if !visit(v.Currency) {
					return
				}
			case Sell:
				if !visit(v.Currency) {
					return
				}
			case Dividend:
				if !visit(v.Currency) {
					return
				}
			case Deposit:
				if !visit(v.Currency) {
					return
				}
			case Withdraw:
				if !visit(v.Currency) {
					return
				}
			case Convert:
				if !visit(v.FromCurrency) {
					return
				}
				if !visit(v.ToCurrency) {
					return
				}
			}
		}
	}
}
