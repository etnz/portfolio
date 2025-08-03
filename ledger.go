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

// Transactions iterates over transaction in their order.
func (l Ledger) Transactions() iter.Seq2[int, Transaction] {
	return func(yield func(int, Transaction) bool) {
		for i, tx := range l.transactions {
			if !yield(i, tx) {
				return
			}
		}
	}
}

// Sort sorts the ledger by transaction date. The sort is stable, meaning
// transactions on the same day maintain their original relative order.
func (l *Ledger) stableSort() {
	sort.SliceStable(l.transactions, func(i, j int) bool {
		return l.transactions[i].When().Before(l.transactions[j].When())
	})
}

// Position computes the total quantity of a security held on a specific date.
func (l *Ledger) Position(ticker string, on date.Date) float64 {
	var quantity float64
	for _, tx := range l.SecurityTransactions(ticker, on) {
		switch v := tx.(type) {
		case Buy:
			quantity += v.Quantity
		case Sell:
			// quantity should not be turned negative for a valid portfolio.
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

// SecurityTransactions iterates over transaction up to 'max' included of relative to security identified by its ticker.
// Security transactions are: buy or sell or dividend.
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
