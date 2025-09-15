package portfolio

import (
	"errors"
	"fmt"
	"iter"
	"log"
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

// Validate checks a transaction for correctness and applies quick fixes where
// applicable (e.g., resolving "sell all"). It returns the validated (and
// potentially modified) transaction or an error detailing any validation failures.
func (l *Ledger) Validate(tx Transaction, reportingCurrency string) (Transaction, error) {
	// For validations that need the state of the portfolio, we compute the balance
	// on the transaction date.
	var err error
	switch v := tx.(type) {
	case Buy:
		err = v.Validate(l)
		tx = v
	case Sell:
		// todo: recomputing the whole journal everytime is expensive.
		var j *Journal
		j, err = newJournal(l, reportingCurrency)
		if err != nil {
			return nil, fmt.Errorf("invalid journal: %w", err)
		}

		balance, e := NewBalance(j, tx.When(), FIFO) // TODO: Make cost basis method configurable
		if e != nil {
			return nil, fmt.Errorf("could not create balance from journal: %w", e)
		}
		err = v.Validate(l, balance)
		tx = v
	case Dividend:
		err = v.Validate(l)
		tx = v
	case Deposit:
		err = v.Validate(l)
		tx = v
	case Withdraw:
		err = v.Validate(l)
		tx = v
	case Convert:
		err = v.Validate(l)
		tx = v
	case Declare:
		err = v.Validate(l)
		tx = v
	case Accrue:
		err = v.Validate(l)
		tx = v
	case UpdatePrice:
		err = v.Validate(l)
		tx = v
	case Split:
		err = v.Validate(l)
		tx = v
	default:
		return tx, fmt.Errorf("unsupported transaction type for validation: %T %v", tx, tx)
	}
	if err != nil {
		return tx, fmt.Errorf("invalid %s transaction on %v: %w", tx.What(), tx.When(), err)
	}
	return tx, nil
}

// UpdateIntraday fetches the latest intraday prices for all securities in the ledger
// from the tradegate provider and updates the ledger with them.
func (l *Ledger) UpdateIntraday() error {
	var newTxs []Transaction
	var errs error

	// Update the EURUSD ticker
	val, err := tradegateLatestEURperUSD()
	if err != nil {
		errs = errors.Join(errs, fmt.Errorf("could not fetch EUR/USD rate: %w", err))
	} else {
		// Tradegate gives EUR per USD, we want USD per EUR for conversion.
		// We create a fake ticker to store this.
		// TODO: This is a hack. Forex rates should be handled more elegantly.
		newTxs = append(newTxs, NewUpdatePrice(Today(), "USDEUR", M(1/val, "EUR")))
	}

	// then update stocks
	for sec := range l.AllSecurities() {
		var latest float64
		var err error

		id := sec.ID()
		if isin, _, mssiErr := id.MSSI(); mssiErr == nil {
			latest, err = tradegateLatest(sec.Ticker(), isin)
		} else if isin, fundErr := id.ISIN(); fundErr == nil {
			latest, err = tradegateLatest(sec.Ticker(), isin)
		} else {
			// Not a public stock/fund, skip.
			continue
		}

		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("could not get intraday for %s: %w", sec.Ticker(), err))
			continue
		}

		var price Money
		switch sec.Currency() {
		case "USD":
			if val != 0 {
				price = M(latest*val, "USD")
			}
		case "EUR":
			price = M(latest, "EUR")
		}

		if !price.IsZero() {
			newTxs = append(newTxs, NewUpdatePrice(Today(), sec.Ticker(), price))
		}
	}
	l.AppendOrUpdate(newTxs...)
	return errs
}

// Append appends transactions to this ledger and maintains the chronological order of transactions.
func (l *Ledger) Append(txs ...Transaction) {
	l.transactions = append(l.transactions, txs...)
	// process security declarations and counterparty account creation.
	l.processTx(txs...)
	// The ledger is not sorted anymore, the journal is.
	l.stableSort() // Ensure the ledger remains sorted after appending
}

// AppendOrUpdate adds a transaction to the ledger. If the transaction is a
// market data update (UpdatePrice or Split) and an entry for the same security
// on the same day already exists, it replaces the old entry. Otherwise, it
// appends the new transaction.
func (l *Ledger) AppendOrUpdate(txs ...Transaction) {
	for _, tx := range txs {
		var replaced bool
		switch newTx := tx.(type) {
		case UpdatePrice:
			for i, existingTx := range l.transactions {
				if oldTx, ok := existingTx.(UpdatePrice); ok && oldTx.When() == newTx.When() {
					// An UpdatePrice for this day already exists. Merge the prices.
					if oldTx.Prices == nil {
						oldTx.Prices = make(map[string]decimal.Decimal)
					}
					for ticker, price := range newTx.Prices {
						if old, existed := oldTx.Prices[ticker]; !existed || !old.Equal(price) {
							log.Printf("%v: update %v price from %s with %s", newTx.Date, ticker, old, price)
							oldTx.Prices[ticker] = price
						}
					}
					l.transactions[i] = oldTx // Update in place.
					replaced = true
					break // Found the right day, no need to check further.
				}
			}
		case Split:
			for i, existingTx := range l.transactions {
				if oldTx, ok := existingTx.(Split); ok &&
					oldTx.When() == newTx.When() &&
					oldTx.Security == newTx.Security {
					// if identical do nothing
					if oldTx.Numerator != newTx.Numerator || oldTx.Denominator != newTx.Denominator {
						log.Printf("%v: update %v split %v/%v with %v/%v", oldTx.Date, oldTx.Security, oldTx.Numerator, oldTx.Denominator, newTx.Numerator, newTx.Denominator)
						l.transactions[i] = newTx // Replace existing
					}
					// we keep replaced to avoid 'append' later on.
					replaced = true
					break
				}
			}
		case Dividend:
			for i, existingTx := range l.transactions {
				if oldTx, ok := existingTx.(Dividend); ok {
					if oldTx.When() == newTx.When() && oldTx.Security == newTx.Security {
						// if identical do nothing
						if !oldTx.Amount.Equal(newTx.Amount) {
							log.Printf("%v: update %v dividend per share %v with %v", oldTx.Date, oldTx.Security, oldTx.Amount, newTx.Amount)
							l.transactions[i] = newTx // Replace existing
						}
						replaced = true
						break
					}
				}
			}
		}

		if !replaced {
			// If no existing transaction was found and replaced, append the new one.
			l.Append(tx)
			log.Printf("%v: append %q %v", tx.When(), tx.What(), tx)
		}
	}
}

func (l *Ledger) processTx(txs ...Transaction) {
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

// LastKnownMarketDataDate scans the ledger in reverse and returns the date of the most
// recent `update-price` or `split` transaction for the given security ticker.
// The boolean will be true if a date was found, otherwise false.
func (l *Ledger) LastKnownMarketDataDate(security string) (Date, bool) {
	// Iterate backwards for efficiency, as we want the most recent date.
	for i := len(l.transactions) - 1; i >= 0; i-- {
		tx := l.transactions[i]

		switch v := tx.(type) {
		case UpdatePrice:
			if _, ok := v.Prices[security]; ok {
				return v.Date, true
			}
		case Split:
			if v.Security == security {
				return v.When(), true
			}
		default:
			// to avoid lint warning
		}
	}
	return Date{}, false
}

// InceptionDate scans the ledger and returns the date of the very first
// transaction of any kind for the given security ticker.
func (l *Ledger) InceptionDate(security string) (Date, bool) {
	for _, tx := range l.transactions {
		var ticker string
		switch v := tx.(type) {
		case Buy:
			ticker = v.Security
		case Sell:
			ticker = v.Security
		case Dividend:
			ticker = v.Security
		case Declare: // ignore declare the date is meaningless.
			ticker = v.Ticker
		case UpdatePrice:
			// An UpdatePrice can have multiple tickers, so we need to check them all.
			for t := range v.Prices {
				if t == security {
					return tx.When(), true
				}
			}
			continue // Skip to next transaction
		case Split:
			ticker = v.Security
		default:
			continue
		}

		if ticker == security {
			return tx.When(), true
		}
	}
	return Date{}, false
}

// Clean remove spurious market data from the ledger
// On a given day, market data relative to an asset not held will be deleted.
func (l *Ledger) Clean() error {
	// creates a journal, and scan it computing correct positions and cleaning the ledger on the fly
	j, err := newJournal(l, "EUR")
	if err != nil {
		return fmt.Errorf("could not create journal: %w", err)
	}

	holdings := make(map[string]Quantity)

	for _, e := range j.events {
		switch v := e.(type) {
		case acquireLot:
			holdings[v.security] = holdings[v.security].Add(v.quantity)
		case disposeLot:
			holdings[v.security] = holdings[v.security].Sub(v.quantity)
		case splitShare:
			if holdings[v.security].IsZero() {
				// mark the source transaction for deletion.
				l.transactions[e.source()] = nil
				continue
			}
			num := Q(v.numerator)
			den := Q(v.denominator)
			holdings[v.security] = holdings[v.security].Mul(num).Div(den)

		case updatePrice:
			if holdings[v.security].IsZero() {
				// modify the source transaction to delete this asset's price.
				u := l.transactions[e.source()].(UpdatePrice)
				delete(u.Prices, v.security)
				if len(u.Prices) == 0 {
					// Delete the source transaction if empty.
					l.transactions[e.source()] = nil
				} else {
					// Otherwise copy the shrinked version.
					l.transactions[e.source()] = u
				}
			}

		case receiveDividend:
			if holdings[v.security].IsZero() {
				// mark the source transaction for deletion.
				l.transactions[e.source()] = nil
				continue
			}
		}
	}
	// Delete marked transactions.
	newTxs := make([]Transaction, 0, len(l.transactions))
	for _, tx := range l.transactions {
		if tx != nil {
			newTxs = append(newTxs, tx)
		}
	}
	l.transactions = newTxs
	return nil
}
