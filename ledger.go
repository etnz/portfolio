package portfolio

import (
	"fmt"
	"iter"
	"maps"
	"slices"

	"sort"

	"github.com/etnz/portfolio/date"
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
	transactions []Transaction
	securities   map[string]Security // index securities by ticker
	// lots is a map from security ticker to a slice of its open lots (purchases not yet fully sold).
	// These lots are used for cost basis calculations (e.g., FIFO, AverageCost) and are kept
	// sorted by date for FIFO accounting.
	lots map[string][]lot
}

// Declared return a iterator over all declared securities in the ledger.
func (l *Ledger) Declared() iter.Seq[Security] {
	// Put securities in a slice.
	securities := slices.Collect(maps.Values(l.securities))
	// Sort securities by ticker.
	sort.Slice(securities, func(i, j int) bool {
		return securities[i].Ticker() < securities[j].Ticker()
	})
	return slices.Values(securities)
}

// NewLedger creates an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{
		transactions: make([]Transaction, 0),
		securities:   make(map[string]Security),
		lots:         make(map[string][]lot),
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

// Append appends transactions to this ledger and maintains the chronological order of transactions.
func (l *Ledger) Append(txs ...Transaction) {
	l.transactions = append(l.transactions, txs...)
	// process security declarations and update lots
	for _, tx := range txs {
		switch v := tx.(type) {
		case Declare:
			sec := NewSecurity(v.ID, v.Ticker, v.Currency)
			l.securities[sec.Ticker()] = sec
		case Buy:
			// Add to lots for cost basis tracking
			l.lots[v.Security] = append(l.lots[v.Security], lot{
				Date:     v.When(),
				Quantity: decimal.NewFromFloat(v.Quantity),
				Cost:     decimal.NewFromFloat(v.Amount),
			})
			// Keep lots sorted by date for FIFO
			sort.Slice(l.lots[v.Security], func(i, j int) bool {
				return l.lots[v.Security][i].Date.Before(l.lots[v.Security][j].Date)
			})
		}
	}

	l.stableSort() // Ensure the ledger remains sorted after appending
}

// Transactions returns an iterator that yields each transaction in its original order.
func (l Ledger) Transactions(filters ...func(Transaction) bool) iter.Seq2[int, Transaction] {
	// The returned iterator preserves the original order of transactions in the ledger.
	return func(yield func(int, Transaction) bool) {
		for i, tx := range l.transactions {
			for _, filter := range filters {
				if !filter(tx) {
					continue
				}
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

// Position computes the total quantity of a security held on a specific date.
// It now requires market data to correctly adjust for any stock splits that
// have occurred.
func (l *Ledger) Position(ticker string, on date.Date, market *MarketData) float64 {
	totalPosition := decimal.NewFromInt(0)
	secID := l.securities[ticker].ID()
	splits := market.Splits(secID)

	for _, tx := range l.SecurityTransactions(ticker, on) {
		var quantity decimal.Decimal
		var txDate date.Date

		switch v := tx.(type) {
		case Buy:
			quantity = decimal.NewFromFloat(v.Quantity)
			txDate = v.When()
		case Sell:
			quantity = decimal.NewFromFloat(v.Quantity).Neg() // Sell is a negative quantity
			txDate = v.When()
		default:
			continue // Not a position-affecting transaction
		}

		quantity = adjustForSplits(quantity, txDate, on, splits)
		totalPosition = totalPosition.Add(quantity)
	}

	pos, _ := totalPosition.Float64()
	return pos
}

func adjustForSplits(quantity decimal.Decimal, txDate date.Date, on date.Date, splits []Split) decimal.Decimal {
	if AdjustedPrices {
		return quantity
	}
	adjustmentFactor := decimal.NewFromInt(1)
	for _, split := range splits {
		// A split applies if it happened AFTER the transaction but ON OR BEFORE the query date
		if split.Date.After(txDate) && !split.Date.After(on) {
			num := decimal.NewFromInt(split.Numerator)
			den := decimal.NewFromInt(split.Denominator)
			splitRatio := num.Div(den)
			adjustmentFactor = adjustmentFactor.Mul(splitRatio)
		}
	}
	return quantity.Mul(adjustmentFactor)
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
				balance -= v.Amount
			}
		case Sell:
			if sec, ok := l.securities[v.Security]; ok && sec.Currency() == currency {
				balance += v.Amount
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

// CounterpartyAccountBalance computes the balance of a counterparty account on a specific date.
func (l *Ledger) CounterpartyAccountBalance(account string, on date.Date) (float64, string) {
	var balance float64
	var currency string
	for _, tx := range l.transactions {
		if tx.When().After(on) {
			break
		}
		switch v := tx.(type) {
		case Accrue:
			if v.Counterparty == account {
				balance += v.Amount
				currency = v.Currency
			}
		case Deposit:
			if v.Settles == account {
				balance -= v.Amount
				currency = v.Currency
			}
		case Withdraw:
			if v.Settles == account {
				balance += v.Amount
				currency = v.Currency
			}
		}
	}
	return balance, currency
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
			case Buy:
				if sec := l.Get(v.Security); sec != nil {
					visitedCurrencies[sec.Currency()] = struct{}{}
				}
			case Sell:
				if sec := l.Get(v.Security); sec != nil {
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
