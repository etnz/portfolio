package portfolio

import (
	"errors"
	"fmt"
	"iter"
	"log"
	"maps"
	"slices"

	"github.com/shopspring/decimal"
)

// Ledger represents a list of transactions.
//
// In a Ledger transactions are always in chronological order.
type Ledger struct {
	name           string
	currency       string // ledger currency
	transactions   []Transaction
	securities     map[string]Security // index securities by ticker
	counterparties map[string]string   // index counterparties currency by counterparty name
	journal        *Journal
}

// NewLedger creates an empty ledger.
func NewLedger() *Ledger {
	return &Ledger{
		currency:       "EUR",
		transactions:   make([]Transaction, 0),
		securities:     make(map[string]Security),
		counterparties: make(map[string]string),
		journal:        &Journal{},
	}
}

// Fmt creates a new, formatted ledger from the current one.
// It validates and sorts all transactions, applying quick fixes where applicable.
// This produces a canonical version of the ledger.
func (l *Ledger) Fmt() (*Ledger, error) {
	// The source ledger needs to be sorted to process transactions in order for validation.
	l.stableSort()

	// Create a new empty ledger, preserving the name and currency from the original.
	newLedger := NewLedger()
	newLedger.name = l.name
	newLedger.currency = l.currency

	// Append transactions one by one to the new ledger. The Append method
	// will handle validation and re-building the internal state (journal).
	for _, tx := range l.transactions {
		validatedTx, err := newLedger.Validate(tx)
		if err != nil {
			return nil, fmt.Errorf("validation failed for transaction on %s (%T): %w", tx.When(), tx, err)
		}
		if err := newLedger.Append(validatedTx); err != nil {
			return nil, fmt.Errorf("failed to append transaction on %s: %w", tx.When(), err)
		}
	}
	return newLedger, nil
}

// Currencies returns a sequence of all currencies used in the ledger as of today.
func (ledger *Ledger) Currencies() iter.Seq[string] {
	return ledger.NewSnapshot(Today()).Currencies()
}

func (ledger *Ledger) Currency() string { return ledger.currency }

// Name returns the name of the ledger, which is its relative path from the portfolio root.
func (ledger *Ledger) Name() string { return ledger.name }

func (l *Ledger) CounterPartyCurrency(account string) (cur string, exists bool) {
	// TODO: ledger should not hold index, this should be the journal.
	cur, ok := l.counterparties[account]
	return cur, ok
}

// Security return the security declared with this ticker, or nil if unknown.
func (l *Ledger) Security(ticker string) *Security {
	// TODO: ledger should not hold index, this should be the journal.
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
	return tx.Validate(l)
}

// UpdateIntraday fetches the latest intraday prices for all securities in the ledger
// from the tradegate provider and updates the ledger with them.
func (l *Ledger) UpdateIntraday() error {
	// TODO: Update Intraday should be done differently.
	// a provider should be made, that can fetch data, and the UpdateMarketData should be used instead.
	var newTxs []Transaction
	var errs error
	today := Today()

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
		if !l.Position(today, sec.Ticker()).IsZero() {
			if isin, _, mssiErr := id.MSSI(); mssiErr == nil {
				latest, err = tradegateLatest(sec.Ticker(), isin)
			} else if isin, fundErr := id.ISIN(); fundErr == nil {
				latest, err = tradegateLatest(sec.Ticker(), isin)
			} else {
				// Not a public stock/fund, skip.
				continue
			}
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
	// TODO: if txs are in order, we can append to the existing journal instead of recomputing it.
	return l.newJournal()
}

// MarketDataUpdate provides a summary of changes made during a market data update.
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

// UpdateMarketData adds transactions to the ledger.
func (l *Ledger) UpdateMarketData(txs ...Transaction) (MarketDataUpdate, error) {

	// Separate the transactions by type because we have to process
	// them in a specific order: Splits, Dividends, UpdatePrice.
	splits := make([]Split, 0)
	updates := make([]UpdatePrice, 0)
	dividends := make([]Dividend, 0)
	for _, t := range txs {
		switch ntx := t.(type) {
		case Split:
			splits = append(splits, ntx) // just to have the right type

		case UpdatePrice:
			// just store for later process
			updates = append(updates, ntx)
		case Dividend:
			dividends = append(dividends, ntx)
		}
	}

	// handle splits first, because splits affects holdings.
	newSplits, updatedSplits := 0, 0
	for _, ntx := range splits {
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
	}

	if newSplits > 0 || updatedSplits > 0 {
		// Splits have changed, journal is now obsolete.
		if err := l.newJournal(); err != nil {
			return MarketDataUpdate{newSplits: newSplits, updatedSplits: updatedSplits}, fmt.Errorf("invalid split transactions: %w", err)
		}
	}

	// Now dividends can be processed, we have the correct holdings.

	addedDiv, updatedDiv := 0, 0
	for _, ndiv := range dividends {
		if l.Position(ndiv.Date, ndiv.Security).IsZero() {
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

	// Append all updatePrice
	addedPrices, updatedPrices := 0, 0
	for _, nup := range updates {
		// Ignore prices related to non existing securities.
		for t := range nup.PricesIter() {
			// Remove prices from the update related to unknown securities.
			if _, exists := l.securities[t]; !exists {
				delete(nup.Prices, t)
			}
		}

		// Skip that update if empty.
		if len(nup.Prices) == 0 {
			continue
		}

		// Find an UpdatePrice for the same day to merge into.
		index, updatePrice := -1, UpdatePrice{}
		for i, tx := range l.transactions {
			prev, isUpdatePrice := tx.(UpdatePrice)
			if isUpdatePrice && prev.When() == nup.When() {
				index, updatePrice = i, prev
				break
			}
		}

		if index < 0 {
			// There were no UpdatePrice for that date. The process is to simply append it.
			v, err := l.Validate(nup)
			if err != nil {
				return MarketDataUpdate{}, fmt.Errorf("invalid update price transactions: %w", err)
			}

			l.transactions = append(l.transactions, v)
			addedPrices += len(nup.Prices)
		} else {
			// There was an pre-existing UpdatePrice for this date, we have to merge them.

			// The goal is to merge new prices into existing ones, overwriting existing prices if needed.
			// But for logging reasons we want to know what was really new.
			// So we split new prices and existing prices into two sets: the merged set, and the really new set.
			onlyNew, all := mergePrices(nup.Prices, updatePrice.Prices)

			// Create a new UpdatePrice transaction with the merged prices
			mergedTx := NewUpdatePrices(nup.When(), all)

			// Update in place the existing UpdatePrice.
			l.transactions[index] = mergedTx
			updatedPrices += len(onlyNew)
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

// processTx processes a slice of transactions to update the ledger's internal
// indexes, such as declared securities and counterparty accounts.
// This is typically called after transactions are appended to the ledger.
func (l *Ledger) processTx(txs ...Transaction) {
	for _, tx := range txs {
		switch v := tx.(type) {
		case Init:
			l.currency = v.Currency
		case Declare:
			sec := NewSecurity(v.ID, v.Ticker, v.Currency, v.Memo)
			l.securities[sec.Ticker()] = sec
		case Accrue:
			if v.Create {
				l.counterparties[v.Counterparty] = v.Amount.cur
			}
		}
	}
}

// Transactions returns an iterator that yields each transaction in their original order with their original index.
func (l Ledger) Transactions(accepts ...func(Transaction) bool) iter.Seq2[int, Transaction] {
	// TODO: there are other function to iterator over transactions, not all use the filter mechanism,
	// we should probably unify them
	// The returned iterator preserves the original order of transactions in the ledger.
	return func(yield func(int, Transaction) bool) {
		for i, tx := range l.transactions {
			acceptOr := false
			for _, accept := range accepts {
				if accept(tx) {
					acceptOr = true
					break
				}
			}
			if !acceptOr {
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
// Some transactions should be put at the beginning of the day:
//   - Declare are the very first ones
//   - Dividend, UpdatePrice, and Splits aka Market Data transactions come second
//   - All other transactions come last.
func (l *Ledger) stableSort() {
	slices.SortStableFunc(l.transactions, func(a, b Transaction) int {
		// we want to sort by Date first, by classes of transaction second.

		// First compute the transactions classes.
		const init, declare, market, ops = 0, 1, 2, 3
		const classes = 4
		classOf := func(t CommandType) int {
			switch t {
			case CmdInit:
				return init
			case CmdDeclare:
				return declare
			case CmdDividend, CmdSplit, CmdUpdatePrice:
				return market
			default:
				return ops
			}
		}
		classA, classB := classOf(a.What()), classOf(b.What())
		dateA, dateB := a.When(), b.When()

		return dateA.Compare(dateB)*classes + classA - classB
	})

}

// GlobalInceptionDate returns the date of the earliest transaction, which should be the Init transaction.
func (l *Ledger) GlobalInceptionDate() Date {
	if len(l.transactions) > 0 {
		return l.transactions[0].When()
	}
	return Date{}
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
	return l.NewSnapshot(on).Cash(currency)
}

// Position computes the current holding for a ticker
func (l *Ledger) Position(on Date, ticker string) Quantity {
	if l.journal == nil {
		return Q(decimal.Zero)
	}
	return l.NewSnapshot(on).Position(ticker)
}

// CounterpartyAccountBalance computes the balance of a counterparty account on a specific date.
func (l *Ledger) CounterpartyAccountBalance(account string, on Date) Money {
	s := l.NewSnapshot(on)
	return s.Counterparty(account)
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

// AcceptAll transactions.
func AcceptAll(Transaction) bool { return true }

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

// ByUpdatePrice returns a predicate that accept only UpdatePrice transactions
func ByUpdatePrice() func(Transaction) bool {
	return func(tx Transaction) bool {
		switch tx.(type) {
		case UpdatePrice:
			return true
		default:
			return false
		}
	}
}

// HeldSecuritiesInRange returns an iterator for all securities that had a non-zero
// position at any point within the given date range.
func (l *Ledger) HeldSecuritiesInRange(period Range) iter.Seq[Security] {
	return func(yield func(Security) bool) {
		heldTickers := make(map[string]struct{})

		// Get securities held at the start of the period.
		startSnap := l.NewSnapshot(period.From.Add(-1))
		for ticker := range l.AllSecurities() {
			if !startSnap.Position(ticker.Ticker()).IsZero() {
				heldTickers[ticker.Ticker()] = struct{}{}
			}
		}

		// Add securities transacted within the period.
		for _, tx := range l.transactions {
			if period.Contains(tx.When()) {
				switch v := tx.(type) {
				case Buy:
					heldTickers[v.Security] = struct{}{}
				case Sell:
					heldTickers[v.Security] = struct{}{}
				}
			}
		}

		// Yield the full security info for each held ticker.
		for ticker := range heldTickers {
			if sec, ok := l.securities[ticker]; ok {
				if !yield(sec) {
					return
				}
			}
		}
	}
}

// LastOperationDate returns the date of the last operation for a given security ticker.
func (ledger *Ledger) LastOperationDate(s string) Date {
	// Iterate backwards for efficiency, as we want the most recent date.
	for i := len(ledger.transactions) - 1; i >= 0; i-- {
		tx := ledger.transactions[i]

		switch v := tx.(type) {
		case Buy:
			if v.Security == s {
				return v.Date
			}
		case Sell:
			if v.Security == s {
				return v.Date
			}
		}
	}
	return Date{}
}

// LastKnownMarketDataDate scans the ledger in reverse and returns the date of the most
// recent `update-price` or `split` transaction for the given security ticker.
func (l *Ledger) LastKnownMarketDataDate(security string) Date {
	// Iterate backwards for efficiency, as we want the most recent date.
	for i := len(l.transactions) - 1; i >= 0; i-- {
		tx := l.transactions[i]

		switch v := tx.(type) {
		case UpdatePrice:
			if _, ok := v.Prices[security]; ok {
				return v.Date
			}
		case Split:
			if v.Security == security {
				return v.When()
			}
		default:
			// to avoid lint warning
		}
	}
	return Date{}
}

// InceptionDate scans the ledger and returns the date of the very first
// transaction of any kind for the given security ticker.
func (l *Ledger) InceptionDate(security string) Date {
	for _, tx := range l.transactions {
		switch v := tx.(type) {
		case Buy:
			if security == v.Security {
				return tx.When()
			}
		case Sell:
			if security == v.Security {
				return tx.When()
			}
		case Dividend:
			if security == v.Security {
				return tx.When()
			}
		case Declare:
			if security == v.Ticker {
				return tx.When()
			}
		case UpdatePrice:
			// An UpdatePrice can have multiple tickers, so we need to check them all.
			for t := range v.Prices {
				if t == security {
					return tx.When()
				}
			}
			continue // Skip to next transaction
		case Split:
			if security == v.Security {
				return tx.When()
			}
		default:
			continue
		}
	}
	return Date{}
}

// NewSnapshot creates a new portfolio snapshot for a given date.
func (l *Ledger) NewSnapshot(on Date) *Snapshot {
	return &Snapshot{
		journal: l.journal,
		on:      on,
	}
}

// NewReview creates a new portfolio review for a given period.
func (l *Ledger) NewReview(period Range) (*Review, error) {
	return NewReview(l.journal, period)
}

// GenerateLog generates a log of reviews for each sub-period within a given date range.
func (l *Ledger) GenerateLog(r Range, period Period) ([]*Review, error) {
	var result []*Review
	if l.journal == nil {
		return nil, errors.New("empty ledger")
	}

	for subRange := range r.Periods(period) {
		review, err := l.NewReview(subRange)
		if err != nil {
			return nil, err
		}
		result = append(result, review)
	}
	return result, nil
}

// Journal returns the ledger's journal.
func (l *Ledger) Journal() *Journal {
	return l.journal
}
