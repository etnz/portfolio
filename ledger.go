package portfolio

import (
	"errors"
	"fmt"
	"iter"
	"log"
	"maps"
	"slices"
	"strings"

	"github.com/shopspring/decimal"
)

// Ledger represents a list of transactions.
//
// In a Ledger transactions are always in chronological order.
type Ledger struct {
	currency       string // ledger currency
	transactions   []Transaction
	securities     map[string]Security // index securities by ticker
	counterparties map[string]string   // index counterparties currency by counterparty name
	journal        *Journal
}

// NewLedger creates an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{
		transactions:   make([]Transaction, 0),
		securities:     make(map[string]Security),
		counterparties: make(map[string]string),
		journal:        nil,
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
func (l *Ledger) Validate(tx Transaction) (Transaction, error) {
	// For validations that need the state of the portfolio, we compute the balance
	// on the transaction date.
	var err error
	switch v := tx.(type) {
	case Buy:
		err = v.Validate(l)
		tx = v
	case Sell:
		err = v.Validate(l)
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
	l.UpdateMarketData(newTxs...)
	return errs
}

// Append appends transactions to this ledger and maintains the chronological order of transactions.
func (l *Ledger) Append(txs ...Transaction) error {
	// logic is a bit more complicated than that.
	l.transactions = append(l.transactions, txs...)
	// process security declarations and counterparty account creation.
	l.processTx(txs...)
	// The ledger is not sorted anymore, the journal is.
	return l.newJournal()
}

type MarketDataUpdate struct {
	newSplits, updatedSplits, addedDiv, updatedDiv, addedPrices, updatedPrices int
}

func (m MarketDataUpdate) NewSplits() int        { return m.newSplits }
func (m MarketDataUpdate) UpdatedSplits() int    { return m.updatedSplits }
func (m MarketDataUpdate) AddedDividends() int   { return m.addedDiv }
func (m MarketDataUpdate) UpdatedDividends() int { return m.updatedDiv }
func (m MarketDataUpdate) AddedPrices() int      { return m.addedPrices }
func (m MarketDataUpdate) UpdatedPrices() int    { return m.updatedPrices }
func (m MarketDataUpdate) Total() int {
	return m.NewSplits() + m.UpdatedSplits() + m.AddedDividends() + m.UpdatedDividends() + m.AddedPrices() + m.UpdatedPrices()
}

// UpdateMarketData adds a transaction to the ledger. If the transaction is a
// market data update (UpdatePrice or Split) and an entry for the same security
// on the same day already exists, it replaces the old entry. Otherwise, it
// appends the new transaction.
func (l *Ledger) UpdateMarketData(txs ...Transaction) (MarketDataUpdate, error) {

	// first thing is to append splits, because they might cause changes in holdings for instance
	// we'll dispatch transactions per type first.

	newSplits, updatedSplits := 0, 0
	updates := make([]UpdatePrice, 0)
	dividends := make([]Dividend, 0)
	for _, t := range txs {
		switch ntx := t.(type) {
		case Split:
			// Find a split same day same security
			index, split := -1, Split{}
			for i, tx := range l.transactions {
				prev, isSplit := tx.(Split)
				if isSplit && prev.Security == ntx.Security && prev.When() == ntx.When() {
					index, split = i, prev
					break
				}
			}
			if index < 0 {
				// Add
				log.Printf("%v: append %v split %v/%v", ntx.Date, ntx.Security, ntx.Numerator, ntx.Denominator)
				newSplits++
				l.transactions = append(l.transactions, ntx)
			} else {
				if split.Numerator != ntx.Numerator || split.Denominator != ntx.Denominator {
					// new is different, update in place.
					log.Printf("%v: add %v split %v/%v -> %v/%v", ntx.Date, ntx.Security, split.Numerator, split.Denominator, ntx.Numerator, ntx.Denominator)
					updatedSplits++
					l.transactions[index] = ntx
				}
			}

		case UpdatePrice:
			// just store for later process
			updates = append(updates, ntx)
		case Dividend:
			dividends = append(dividends, ntx)
		}
	}

	if newSplits > 0 || updatedSplits > 0 {
		// Splits have changed, journal is not obsolete.
		if err := l.newJournal(); err != nil {
			return MarketDataUpdate{newSplits: newSplits, updatedSplits: updatedSplits}, fmt.Errorf("invalid split transactions: %w", err)
		}
	}
	// now we have a pretty nice ledger to work with.

	// append all dividends
	addedDiv, updatedDiv := 0, 0
	for _, ndiv := range dividends {
		if l.Holding(ndiv.Date, ndiv.Security).IsZero() {
			continue // skip dividends on non held assets.
		}
		index, div := -1, Dividend{}
		for i, tx := range l.transactions {
			prev, isDiv := tx.(Dividend)
			if isDiv && prev.Security == ndiv.Security && prev.When() == ndiv.When() {
				index, div = i, prev
				break
			}
		}
		if index < 0 {
			// Add
			log.Printf("%v: add %v dividend %v", ndiv.Date, ndiv.Security, ndiv.Amount)
			l.transactions = append(l.transactions, ndiv)
			addedDiv++
		} else {
			if !div.Amount.Equal(ndiv.Amount) {
				// new is different, update in place.
				log.Printf("%v: update %v dividend %v -> %v", ndiv.Date, ndiv.Security, div.Amount, ndiv.Amount)
				updatedDiv++
				l.transactions[index] = ndiv
			}
		}
	}

	// append all updatePrice
	addedPrices, updatedPrices := 0, 0
	for _, nup := range updates {
		// remove updates relative to non held
		for t := range nup.PricesIter() {
			if l.Holding(nup.Date, t).IsZero() {
				// remove it from the updates
				delete(nup.Prices, t)
			}
		}
		if len(nup.Prices) == 0 {
			continue
		}
		// Check if an UpdatePrice transaction for the same date already exists.
		index, updatePrice := -1, UpdatePrice{}
		for i, tx := range l.transactions {
			prev, isUpdatePrice := tx.(UpdatePrice)
			if isUpdatePrice && prev.When() == nup.When() {
				index, updatePrice = i, prev
				break
			}
		}

		if index < 0 {
			// No existing UpdatePrice for this date, append the new one.
			var buf strings.Builder
			for ticker, price := range nup.PricesIter() {
				buf.WriteString(ticker)
				buf.WriteString(":")
				buf.WriteString(price.String())
				buf.WriteString(" ")
			}

			log.Printf("%v: add update-price %v", nup.Date, buf.String())
			l.transactions = append(l.transactions, nup)
			addedPrices += len(nup.Prices)
		} else {
			// clean existing from updates relative to non held assets
			for t := range updatePrice.PricesIter() {
				if l.Holding(updatePrice.Date, t).IsZero() {
					// remove it from the updates
					delete(updatePrice.Prices, t)
				}
			}
			// We have a bunch of newprices for some tickers and another bunch of existing prices.
			// some of the 'new' are not new (same value), and some of the existing ones need to be kept.
			// we want to count the really new ones (to actually change the ledger)
			// we want to merge all prices (new, and old), new having priority.
			onlyNew, all := mergePrices(nup.Prices, updatePrice.Prices)

			addedPrices += len(onlyNew)
			nup.Prices = all

			if len(onlyNew) > 0 {
				var buf strings.Builder
				for ticker, price := range onlyNew {
					buf.WriteString(ticker)
					buf.WriteString(":")
					buf.WriteString(price.String())
					buf.WriteString(" ")
				}
				log.Printf("%v: update existing update-price %v", nup.Date, buf.String())
				l.transactions[index] = nup
				updatedPrices++
			}
		}
	}

	upd := MarketDataUpdate{newSplits: newSplits, updatedSplits: updatedSplits, addedDiv: addedDiv, updatedDiv: updatedDiv, addedPrices: addedPrices, updatedPrices: updatedPrices}

	if addedPrices > 0 || updatedPrices > 0 || addedDiv > 0 || updatedDiv > 0 {
		// If any market data was added or updated, the journal is now obsolete.
		if err := l.newJournal(); err != nil {
			return upd, fmt.Errorf("invalid market data transactions: %w", err)
		}
	}

	return upd, nil
}

// We have a bunch of newprices for some tickers and another bunch of existing prices.
// some of the 'new' are not new (same value), and some of the existing ones need to be kept.
// we want to count the really new ones (to actually change the ledger)
// we want to merge all prices (new, and old), new having priority.
func mergePrices(updated, existing map[string]decimal.Decimal) (onlyNew, all map[string]decimal.Decimal) {
	all = make(map[string]decimal.Decimal)
	maps.Copy(all, existing)
	maps.Copy(all, updated)
	// now compute the only new one
	onlyNew = make(map[string]decimal.Decimal)
	for k, v := range updated {
		e, existed := existing[k]
		if !existed || !e.Equal(v) {
			onlyNew[k] = v
		}
	}
	return onlyNew, all
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
//
// Some transactions should be put at the beginning of the day: Dividend, UpdatePrice, and Splits aka Market Data transactions.
func (l *Ledger) stableSort() {
	slices.SortStableFunc(l.transactions, func(a, b Transaction) int {
		// we want to sort by year, month, day, class of transaction.
		dateA, dateB := a.When(), b.When()
		// return -1 or +1 is correct, return a reasonable distance makes it faster.
		// we'll use 12 month in a year, 30 days in a month, and 2 classes
		const months, days, classes = 12, 30, 3

		if dateA.y != dateB.y {
			return (dateA.y - dateB.y) * months * days * classes
		}
		if dateA.m != dateB.m {
			return int(dateA.m-dateB.m) * days * classes
		}
		if dateA.d != dateB.d {
			return (dateA.d - dateB.d) * classes
		}
		const declare, market, ops = 0, 1, 2
		classOf := func(t CommandType) int {
			switch t {
			case CmdDeclare:
				return declare
			case CmdDividend, CmdSplit, CmdUpdatePrice:
				return market
			default:
				return ops
			}
		}
		classA, classB := classOf(a.What()), classOf(b.What())
		if classA != classB {
			return classA - classB
		}
		return 0 // they are identical
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
	if l.journal == nil {
		return M(decimal.Zero, currency)
	}
	return l.journal.CashBalance(on, currency)
}

// Holding computes the current holding for a ticker
func (l *Ledger) Holding(on Date, ticker string) Quantity {
	if l.journal == nil {
		return Q(decimal.Zero)
	}
	return l.journal.Holding(ticker, on)
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
	if l.journal != nil {
		return nil
	}
	j := l.journal

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
