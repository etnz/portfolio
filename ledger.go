package portfolio

import (
	"iter"
	"maps"
	"slices"

	"sort"

	"github.com/etnz/portfolio/date"
)

// Ledger represents a list of transactions.
//
// In a Ledger transactions are always in chronological order.
type Ledger struct {
	transactions []Transaction
	securities   map[string]Security // index securities by ticker
}

// NewLedger creates an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{
		transactions: make([]Transaction, 0),
		securities:   make(map[string]Security),
	}
}

// Get return the security declared with this ticker, or nil if unknown.
func (l *Ledger) Get(ticker string) *Security {
	sec, ok := l.securities[ticker]
	if !ok {
		return nil
	}
	return &sec
}

// Append appends transactions to this ledger.
func (l *Ledger) Append(txs ...Transaction) {
	l.transactions = append(l.transactions, txs...)
	// process security declarations
	for _, tx := range txs {
		if dec, ok := tx.(Declare); ok {
			sec := NewSecurity(dec.ID, dec.Ticker, dec.Currency)
			l.securities[sec.Ticker()] = sec
		}
	}

	l.stableSort() // Ensure the ledger remains sorted after appending
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
			if sec, ok := l.securities[v.Security]; ok && sec.Currency() == currency {
				balance -= v.Quantity * v.Price
			}
		case Sell:
			if sec, ok := l.securities[v.Security]; ok && sec.Currency() == currency {
				balance += v.Quantity * v.Price
			}
		case Dividend:
			if sec, ok := l.securities[v.Security]; ok && sec.Currency() == currency {
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

// AllSecurities iterates over securities declared in this ledger.
func (l *Ledger) AllSecurities() iter.Seq[Security] {
	return func(yield func(Security) bool) {
		tickers := slices.Collect(maps.Keys(l.securities))
		slices.Sort(tickers)
		for _, ticker := range tickers {
			if !yield(l.securities[ticker]) {
				return
			}
		}
	}
}

// AllCurrencies iterates over all currencies that appear in the ledger.
// in the ledger's transactions.
func (l *Ledger) AllCurrencies() iter.Seq[string] {
	return func(yield func(string) bool) {

		visitedCurrencies := make(map[string]struct{})
		// first visit all, then yeild
		for _, tx := range l.transactions {
			switch v := tx.(type) {
			case Deposit:
				visitedCurrencies[v.Currency] = struct{}{}
			case Withdraw:
				visitedCurrencies[v.Currency] = struct{}{}
			case Convert:
				visitedCurrencies[v.FromCurrency] = struct{}{}
				visitedCurrencies[v.ToCurrency] = struct{}{}
			case Declare:
				visitedCurrencies[v.Currency] = struct{}{}
			}
		}
		// Now yield the values
		for currency := range visitedCurrencies {
			if !yield(currency) {
				return
			}
		}
	}
}
