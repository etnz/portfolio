package portfolio

import (
	"fmt"
	"sort"

	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

// event represents a single, atomic operation in the portfolio's history.
// It is the lowest-level, immutable fact from which all states are derived.
type event interface {
	date() date.Date
}

// Journal holds a chronologically sorted list of all atomic events.
type Journal struct {
	cur    string // the reporting currency.
	events []event
}

// --- Cash Events ---

// creditCash increases the balance of a cash account.
type creditCash struct {
	on       date.Date
	currency string
	amount   decimal.Decimal
}

func (e creditCash) date() date.Date { return e.on }

// debitCash decreases the balance of a cash account.
type debitCash struct {
	on       date.Date
	currency string
	amount   decimal.Decimal
}

func (e debitCash) date() date.Date { return e.on }

// --- Security Events ---

// acquireLot adds a new lot of a security.
type acquireLot struct {
	on       date.Date
	security string
	quantity decimal.Decimal
	cost     decimal.Decimal
}

func (e acquireLot) date() date.Date { return e.on }

// disposeLot removes a quantity of a security.
type disposeLot struct {
	on       date.Date
	security string
	quantity decimal.Decimal
	proceeds decimal.Decimal
}

func (e disposeLot) date() date.Date { return e.on }

// --- Counterparty Events ---

// declareCounterparty maps a ticker to a security ID and currency.
type declareCounterparty struct {
	on       date.Date
	account  string
	currency string
}

func (e declareCounterparty) date() date.Date { return e.on }

// creditCounterparty increases an asset (receivable) or reduces a liability (payable).
type creditCounterparty struct {
	on       date.Date
	account  string
	currency string
	amount   decimal.Decimal
}

func (e creditCounterparty) date() date.Date { return e.on }

// debitCounterparty decreases an asset (receivable) or increases a liability (payable).
type debitCounterparty struct {
	on       date.Date
	account  string
	currency string
	amount   decimal.Decimal
}

func (e debitCounterparty) date() date.Date { return e.on }

// --- Market and Metadata Events ---

// splitShare adjusts the quantity of existing lots for a security.
type splitShare struct {
	on          date.Date
	security    string
	numerator   int64
	denominator int64
}

func (e splitShare) date() date.Date { return e.on }

// declareSecurity maps a ticker to a security ID and currency.
type declareSecurity struct {
	on       date.Date
	ticker   string
	id       ID
	currency string
}

func (e declareSecurity) date() date.Date { return e.on }

// updatePrice sets the price of a security on a given date.
type updatePrice struct {
	on       date.Date
	security string
	price    decimal.Decimal
}

func (e updatePrice) date() date.Date { return e.on }

// updateForex sets the price of a security on a given date.
type updateForex struct {
	on   date.Date
	cur  string // currency rate expressed in the reporting currency
	rate decimal.Decimal
}

func (e updateForex) date() date.Date { return e.on }

// newJournal converts a Ledger of high-level transactions and market data events
// into a Journal of low-level, atomic events.
func (as *AccountingSystem) newJournal() (*Journal, error) {
	journal := &Journal{
		events: make([]event, 0, len(as.Ledger.transactions)*2), // Pre-allocate with a guess
		cur:    as.ReportingCurrency,
	}

	for _, tx := range as.Ledger.transactions {
		switch v := tx.(type) {
		case Buy:
			sec := as.Ledger.Get(v.Security)
			if sec == nil {
				return nil, fmt.Errorf("security %q not declared for buy transaction on %s", v.Security, v.When())
			}
			journal.events = append(journal.events,
				acquireLot{on: v.When(), security: v.Security, quantity: decimal.NewFromFloat(v.Quantity), cost: decimal.NewFromFloat(v.Amount)},
				debitCash{on: v.When(), currency: sec.Currency(), amount: decimal.NewFromFloat(v.Amount)},
			)
		case Sell:
			sec := as.Ledger.Get(v.Security)
			if sec == nil {
				return nil, fmt.Errorf("security %q not declared for sell transaction on %s", v.Security, v.When())
			}
			journal.events = append(journal.events,
				disposeLot{on: v.When(), security: v.Security, quantity: decimal.NewFromFloat(v.Quantity), proceeds: decimal.NewFromFloat(v.Amount)},
				creditCash{on: v.When(), currency: sec.Currency(), amount: decimal.NewFromFloat(v.Amount)},
			)
		case Dividend:
			sec := as.Ledger.Get(v.Security)
			if sec == nil {
				return nil, fmt.Errorf("security %q not declared for dividend transaction on %s", v.Security, v.When())
			}
			journal.events = append(journal.events,
				creditCash{on: v.When(), currency: sec.Currency(), amount: decimal.NewFromFloat(v.Amount)},
			)
		case Deposit:
			amount := decimal.NewFromFloat(v.Amount)
			journal.events = append(journal.events,
				creditCash{on: v.When(), currency: v.Currency, amount: amount},
			)
			if v.Settles != "" {
				// A deposit settling an account means a counterparty paid us back, reducing what they owe us (asset).
				journal.events = append(journal.events,
					debitCounterparty{on: v.When(), account: v.Settles, currency: v.Currency, amount: amount},
				)
			}
		case Withdraw:
			amount := decimal.NewFromFloat(v.Amount)
			journal.events = append(journal.events,
				debitCash{on: v.When(), currency: v.Currency, amount: amount},
			)
			if v.Settles != "" {
				// A withdrawal settling an account means we paid a counterparty back, reducing what we owe them (liability).
				journal.events = append(journal.events,
					creditCounterparty{on: v.When(), account: v.Settles, currency: v.Currency, amount: amount},
				)
			}
		case Convert:
			journal.events = append(journal.events,
				debitCash{on: v.When(), currency: v.FromCurrency, amount: decimal.NewFromFloat(v.FromAmount)},
				creditCash{on: v.When(), currency: v.ToCurrency, amount: decimal.NewFromFloat(v.ToAmount)},
			)
		case Declare:
			journal.events = append(journal.events,
				declareSecurity{on: v.When(), ticker: v.Ticker, id: v.ID, currency: v.Currency},
			)
		case Accrue:
			if v.Create {
				journal.events = append(journal.events, declareCounterparty{on: v.When(), account: v.Counterparty, currency: v.Currency})
			}
			amount := decimal.NewFromFloat(v.Amount)
			if amount.IsPositive() { // Receivable: counterparty owes us (asset) -> increase asset
				journal.events = append(journal.events,
					creditCounterparty{on: v.When(), account: v.Counterparty, currency: v.Currency, amount: amount},
				)
			} else { // Payable: we owe counterparty (liability) -> increase liability
				journal.events = append(journal.events,
					debitCounterparty{on: v.When(), account: v.Counterparty, currency: v.Currency, amount: amount.Neg()},
				)
			}
		default:
			return nil, fmt.Errorf("unhandled transaction type: %T", tx)
		}
	}

	// Add market events like splits and prices.

	// Map market data ID to delcared securities in the ledger.
	idToTickers := make(map[ID][]string)
	for ticker, sec := range as.Ledger.securities {
		idToTickers[sec.ID()] = append(idToTickers[sec.ID()], ticker)
	}

	for id, splits := range as.MarketData.splits {
		tickers := idToTickers[id]
		for _, ticker := range tickers {
			for _, s := range splits {
				journal.events = append(journal.events,
					splitShare{on: s.Date, security: ticker, numerator: s.Numerator, denominator: s.Denominator},
				)
			}
		}
	}

	// UpdatePrice of security declared in the ledger.
	for id, history := range as.MarketData.prices {
		tickers := idToTickers[id]
		for _, ticker := range tickers {
			for on, price := range history.Values() {
				journal.events = append(journal.events,
					updatePrice{on: on, security: ticker, price: decimal.NewFromFloat(price)},
				)
			}
		}
	}

	// UpdateForex update currency forex rate into the reporting one.
	for id, history := range as.MarketData.prices {
		base, quote, err := id.CurrencyPair()
		if err != nil {
			// not a forex
			continue
		}
		switch journal.cur {
		case quote:
			c := base
			for on, price := range history.Values() {
				journal.events = append(journal.events,
					updateForex{on: on, cur: c, rate: decimal.NewFromFloat(price)},
				)
			}
		case base:
			c := quote
			// Provide the reverse forex.
			for on, price := range history.Values() {
				journal.events = append(journal.events,
					updateForex{on: on, cur: c, rate: decimal.NewFromInt(1).Div(decimal.NewFromFloat(price))},
				)
			}
		}
	}

	// Sort all events chronologically.
	sort.SliceStable(journal.events, func(i, j int) bool {
		return journal.events[i].date().Before(journal.events[j].date())
	})

	return journal, nil
}
