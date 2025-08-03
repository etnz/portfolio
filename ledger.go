package portfolio

import (
	"iter"
	"sort"
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
