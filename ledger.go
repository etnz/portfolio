package portfolio

import (
	"fmt"
	"iter"
	"maps"
	"slices"
	"sort"

	"github.com/shopspring/decimal"
)

// CostBasisMethod defines the method for calculating cost basis.
type CostBasisMethod int

const (
	// AverageCost calculates the cost basis by averaging the cost of all shares.
	AverageCost CostBasisMethod = iota
	// FIFO (First-In, First-Out) calculates the cost basis by assuming the first shares purchased are the first ones sold.
	FIFO
)

func (m CostBasisMethod) String() string {
	switch m {
	case AverageCost:
		return "average"
	case FIFO:
		return "fifo"
	default:
		return "unknown"
	}
}

// ParseCostBasisMethod parses a string into a CostBasisMethod.
func ParseCostBasisMethod(s string) (CostBasisMethod, error) {
	switch s {
	case "average":
		return AverageCost, nil
	case "fifo":
		return FIFO, nil
	default:
		return 0, fmt.Errorf("unknown cost basis method: %q", s)
	}
}

// Ledger represents a list of transactions.
//
// In a Ledger transactions are always in chronological order.
type Ledger struct {
	transactions   []Transaction
	securities     map[string]Security // index securities by ticker
	counterparties map[string]string   // index counterparties currency by counterparty name
}

// NewLedger creates an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{
		transactions:   make([]Transaction, 0),
		securities:     make(map[string]Security),
		counterparties: make(map[string]string),
	}
}

func (l *Ledger) CounterPartyCurrency(account string) (cur string, exists bool) {
	cur, ok := l.counterparties[account]
	return cur, ok
}

// Security return the security declared with this ticker, or nil if unknown.
func (l *Ledger) Security(ticker string) *Security {
	sec, ok := l.securities[ticker]
	if !ok {
		return nil
	}
	return &sec
}

// Append appends transactions to this ledger and maintains the chronological order of transactions.
func (l *Ledger) Append(txs ...Transaction) {
	l.transactions = append(l.transactions, txs...)
	// process security declarations and counterparty account creation.
	for _, tx := range txs {
		switch v := tx.(type) {
		case Declare:
			sec := NewSecurity(v.ID, v.Ticker, v.Currency)
			l.securities[sec.Ticker()] = sec
		case Accrue:
			if v.Create {
				l.counterparties[v.Counterparty] = v.Amount.cur
			}
		}
	}
	// The ledger is not sorted anymore, the journal is.
	l.stableSort() // Ensure the ledger remains sorted after appending
}

// Transactions returns an iterator that yields each transaction in its original order.
func (l Ledger) Transactions(filters ...func(Transaction) bool) iter.Seq2[int, Transaction] {
	// The returned iterator preserves the original order of transactions in the ledger.
	return func(yield func(int, Transaction) bool) {
		for i, tx := range l.transactions {
			accept := false
			for _, filter := range filters {
				if filter(tx) {
					accept = true
					break
				}
			}
			if !accept {
				continue
			}
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

// OldestTransactionDate returns the date of the earliest transaction in the ledger.
// It returns false if the ledger has no transactions.
func (l *Ledger) OldestTransactionDate() Date {
	if len(l.transactions) == 0 { // The ledger is not sorted anymore
		return Date{}
	}
	return l.transactions[0].When() // The ledger is not sorted anymore
}

// NewestTransactionDate returns the date of the latest transaction in the ledger.
// It returns false if the ledger has no transactions.
func (l *Ledger) NewestTransactionDate() Date {
	if len(l.transactions) == 0 { // The ledger is not sorted anymore
		return Date{}
	}
	return l.transactions[len(l.transactions)-1].When() // The ledger is not sorted anymore
}

// CashBalance computes the total cash in a specific currency on a specific date.
func (l *Ledger) CashBalance(currency string, on Date) Money {
	balance := M(decimal.Zero, currency)
	for _, tx := range l.transactions {
		if tx.When().After(on) {
			// The ledger is sorted by date, so it's safe to break.
			break
		}
		switch v := tx.(type) {
		case Buy:
			if sec, ok := l.securities[v.Security]; ok && sec.Currency() == currency {
				balance = balance.Sub(v.Amount)
			}
		case Sell:
			if sec, ok := l.securities[v.Security]; ok && sec.Currency() == currency {
				balance = balance.Add(v.Amount)
			}
		case Dividend:
			if sec, ok := l.securities[v.Security]; ok && sec.Currency() == currency {
				balance = balance.Add(v.Amount)
			}
		case Deposit:
			if v.Currency() == currency {
				balance = balance.Add(v.Amount)
			}
		case Withdraw:
			if v.Currency() == currency {
				balance = balance.Sub(v.Amount)
			}
		case Convert:
			// Fix: Use decimal.Decimal's Sub and Add methods for Convert transaction amounts
			if v.FromCurrency() == currency {
				balance = balance.Sub(v.FromAmount)
			}
			if v.ToCurrency() == currency {
				balance = balance.Add(v.ToAmount)
			}
		}
	}
	return balance
}

// CounterpartyAccountBalance computes the balance of a counterparty account on a specific date.
func (l *Ledger) CounterpartyAccountBalance(account string, on Date) Money {
	var balance Money

	for _, tx := range l.transactions {
		if tx.When().After(on) {
			break
		}
		switch v := tx.(type) {
		case Accrue:
			if v.Counterparty == account {
				balance = balance.Add(v.Amount)
			}
		case Deposit:
			if v.Settles == account {
				balance = balance.Add(v.Amount)
			}
		case Withdraw:
			if v.Settles == account {
				balance = balance.Add(v.Amount)
			}

		}
	}
	return balance
}

// AllCounterpartyAccounts returns a sequence of all unique counterparty account names.
func (l *Ledger) AllCounterpartyAccounts() iter.Seq[string] {
	return func(yield func(string) bool) {
		visited := make(map[string]struct{})
		for _, tx := range l.transactions {
			switch v := tx.(type) {
			case Accrue:
				if _, ok := visited[v.Counterparty]; !ok {
					visited[v.Counterparty] = struct{}{}
					if !yield(v.Counterparty) {
						return
					}
				}
			case Deposit:
				if v.Settles != "" {
					if _, ok := visited[v.Settles]; !ok {
						visited[v.Settles] = struct{}{}
						if !yield(v.Settles) {
							return
						}
					}
				}
			case Withdraw:
				if v.Settles != "" {
					if _, ok := visited[v.Settles]; !ok {
						visited[v.Settles] = struct{}{}
						if !yield(v.Settles) {
							return
						}
					}
				}
			}
		}
	}
}

// SecurityTransactions returns an iterator over transactions related to a specific
// security (Buy, Sell, Dividend) up to and including a given date.
// The ledger must be sorted by date for this to work correctly.
func (l Ledger) SecurityTransactions(ticker string, max Date) iter.Seq2[int, Transaction] {
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
				visitedCurrencies[v.Currency()] = struct{}{}
			case Withdraw:
				visitedCurrencies[v.Currency()] = struct{}{}
			case Convert:
				visitedCurrencies[v.FromCurrency()] = struct{}{}
				visitedCurrencies[v.ToCurrency()] = struct{}{}
			case Declare:
				visitedCurrencies[v.Currency] = struct{}{}
			case Buy:
				if sec := l.Security(v.Security); sec != nil {
					visitedCurrencies[sec.Currency()] = struct{}{}
				}
			case Sell:
				if sec := l.Security(v.Security); sec != nil {
					visitedCurrencies[sec.Currency()] = struct{}{}
				}
			}
		}
		// Now yield the values
		currencies := slices.Collect(maps.Keys(visitedCurrencies))
		slices.Sort(currencies)
		for _, currency := range currencies {
			if !yield(currency) {
				return
			}
		}
	}
}

// BySecurity returns a predicate that filters transactions by security ticker.
func BySecurity(ticker string) func(Transaction) bool {
	return func(tx Transaction) bool {
		switch v := tx.(type) {
		case Buy:
			return v.Security == ticker
		case Sell:
			return v.Security == ticker
		case Dividend:
			return v.Security == ticker
		case Declare:
			return v.Ticker == ticker
		default:
			return false
		}
	}
}

// ByCurrency returns a predicate that filters transactions by currency.
func (l *Ledger) ByCurrency(currency string) func(Transaction) bool {
	return func(tx Transaction) bool {
		switch v := tx.(type) {
		case Buy:
			sec := l.Security(v.Security)
			return sec != nil && sec.Currency() == currency
		case Sell:
			sec := l.Security(v.Security)
			return sec != nil && sec.Currency() == currency
		case Dividend:
			sec := l.Security(v.Security)
			return sec != nil && sec.Currency() == currency
		case Deposit:
			return v.Currency() == currency
		case Withdraw:
			return v.Currency() == currency
		case Convert:
			return v.FromCurrency() == currency || v.ToCurrency() == currency
		case Declare:
			return v.Currency == currency
		default:
			return false
		}
	}
}
