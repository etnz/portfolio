package portfolio

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// event represents a single, atomic operation in the portfolio's history.
// It is the lowest-level, immutable fact from which all states are derived.
type event interface {
	date() Date
}

// Journal holds a chronologically sorted list of all atomic events.
type Journal struct {
	cur    string  // the reporting currency.
	events []event // sorted by date
}

// --- Cash Events ---

// creditCash increases the balance of a cash account.
type creditCash struct {
	on       Date
	amount   Money
	external bool // true when cash comes from outside.
}

func (e creditCash) date() Date       { return e.on }
func (e creditCash) currency() string { return e.amount.Currency() }

// debitCash decreases the balance of a cash account.
type debitCash struct {
	on       Date
	amount   Money
	external bool // true when cash goes outside.
}

func (e debitCash) date() Date       { return e.on }
func (e debitCash) currency() string { return e.amount.Currency() }

// --- Security Events ---

// acquireLot adds a new lot of a security.
type acquireLot struct {
	on       Date
	security string
	quantity Quantity
	cost     Money
}

func (e acquireLot) date() Date { return e.on }

// disposeLot removes a quantity of a security.
type disposeLot struct {
	on       Date
	security string
	quantity Quantity
	proceeds Money
}

func (e disposeLot) date() Date { return e.on }

// --- Counterparty Events ---

// declareCounterparty maps a ticker to a security ID and currency.
type declareCounterparty struct {
	on       Date
	account  string
	currency string
}

func (e declareCounterparty) date() Date { return e.on }

// creditCounterparty increases an asset (receivable) or reduces a liability (payable).
type creditCounterparty struct {
	on       Date
	account  string
	amount   Money
	external bool // true when money goes outside.

}

func (e creditCounterparty) date() Date       { return e.on }
func (e creditCounterparty) currency() string { return e.amount.Currency() }

// debitCounterparty decreases an asset (receivable) or increases a liability (payable).
type debitCounterparty struct {
	on       Date
	account  string
	amount   Money
	external bool // true when money goes outside.
}

func (e debitCounterparty) date() Date       { return e.on }
func (e debitCounterparty) currency() string { return e.amount.Currency() }

// --- Market and Metadata Events ---

// splitShare adjusts the quantity of existing lots for a security.
type splitShare struct {
	on          Date
	security    string
	numerator   int64
	denominator int64
}

func (e splitShare) date() Date { return e.on }

// declareSecurity maps a ticker to a security ID and currency.
type declareSecurity struct {
	on       Date
	ticker   string
	id       ID
	currency string
}

func (e declareSecurity) date() Date { return e.on }

// updatePrice sets the price of a security on a given date.
type updatePrice struct {
	on       Date
	security string
	price    Money
}

func (e updatePrice) date() Date { return e.on }

// updateForex sets the price of a security on a given date.
type updateForex struct {
	on       Date
	currency string // the foreign currency (USD in USDEUR)
	rate     Money  //(the cost of 1 USD in EUR (for USDEUR forex))
}

func (e updateForex) date() Date { return e.on }

// NewJournal converts a Ledger of high-level transactions and market data events
// into a Journal of low-level, atomic events.
func newJournal(ledger *Ledger, reportingCurrency string) (*Journal, error) {
	journal := &Journal{
		events: make([]event, 0, len(ledger.transactions)*2), // Pre-allocate with a guess
		cur:    reportingCurrency,
	}

	for _, tx := range ledger.transactions {
		switch v := tx.(type) {
		case Buy:
			sec := ledger.Security(v.Security)
			if sec == nil {
				return nil, fmt.Errorf("security %q not declared for buy transaction on %s", v.Security, v.When())
			}

			journal.events = append(journal.events,
				acquireLot{on: v.When(), security: v.Security, quantity: v.Quantity, cost: v.Amount},
				debitCash{on: v.When(), amount: v.Amount, external: false},
			)
		case Sell:
			sec := ledger.Security(v.Security)
			if sec == nil {
				return nil, fmt.Errorf("security %q not declared for sell transaction on %s", v.Security, v.When())
			}
			journal.events = append(journal.events,
				disposeLot{on: v.When(), security: v.Security, quantity: v.Quantity, proceeds: v.Amount},
				creditCash{on: v.When(), amount: v.Amount, external: false},
			)
		case Dividend:
			sec := ledger.Security(v.Security)
			if sec == nil {
				return nil, fmt.Errorf("security %q not declared for dividend transaction on %s", v.Security, v.When())
			}
			journal.events = append(journal.events,
				creditCash{on: v.When(), amount: v.Amount},
			)
		case Deposit:
			amount := v.Amount
			// A deposit that settles a receivable is not considered as external (since the amount)
			// was taken into account when accruing the receivable
			ext := v.Settles == ""
			journal.events = append(journal.events,
				creditCash{on: v.When(), amount: amount, external: ext},
			)
			if v.Settles != "" {
				// A deposit settling an account means a counterparty paid us back, reducing what they owe us (asset).
				journal.events = append(journal.events,
					debitCounterparty{on: v.When(), account: v.Settles, amount: amount},
				)
			}
		case Withdraw:
			amount := v.Amount
			// A withdrawal that settles a payable is not considered as external (since the amount)
			// was taken into account when accruing the receivable
			ext := v.Settles == ""
			journal.events = append(journal.events,
				debitCash{on: v.When(), amount: amount, external: ext},
			)
			if v.Settles != "" {
				// A withdrawal settling an account means we paid a counterparty back, reducing what we owe them (liability).
				journal.events = append(journal.events,
					creditCounterparty{on: v.When(), account: v.Settles, amount: amount},
				)
			}
		case Convert:
			journal.events = append(journal.events,
				debitCash{on: v.When(), amount: v.FromAmount},
				creditCash{on: v.When(), amount: v.ToAmount},
			)
		case Declare:
			if _, _, err := v.ID.CurrencyPair(); err == nil {
				continue // do not declare currency as securities
			}

			journal.events = append(journal.events,
				declareSecurity{on: v.When(), ticker: v.Ticker, id: v.ID, currency: v.Currency},
			)
		case Accrue:
			if v.Create {
				journal.events = append(journal.events, declareCounterparty{on: v.When(), account: v.Counterparty, currency: v.Currency()})
			}
			amount := v.Amount
			if amount.IsPositive() { // Receivable: counterparty owes us (asset) -> increase asset
				journal.events = append(journal.events,
					creditCounterparty{on: v.When(), account: v.Counterparty, amount: amount, external: true},
				)
			} else { // Payable: we owe counterparty (liability) -> increase liability
				journal.events = append(journal.events,
					debitCounterparty{on: v.When(), account: v.Counterparty, amount: amount.Neg(), external: true},
				)
			}
		case UpdatePrice:
			for ticker, priceDecimal := range v.PricesIter() {
				sec := ledger.Security(ticker)
				if sec == nil {
					// This should have been caught by validation, but we check again.
					return nil, fmt.Errorf("security %q from update-price not declared", ticker)
				}
				price := M(priceDecimal, sec.Currency())

				// Handle forex updates
				if base, quote, err := sec.ID().CurrencyPair(); err == nil {
					if quote == reportingCurrency {
						journal.events = append(journal.events,
							updateForex{on: v.When(), currency: base, rate: price},
						)
					}
					if base == reportingCurrency {
						p := M(decimal.NewFromInt(1).Div(price.value), base)
						p.value = p.value.Round(5) // is enought for an approximate price anyway.
						journal.events = append(journal.events,
							updateForex{on: v.When(), currency: quote, rate: p},
						)
					}
					continue
				}

				// Handle regular security price updates
				journal.events = append(journal.events,
					updatePrice{on: v.When(), security: ticker, price: price},
				)
			}

		case Split:
			journal.events = append(journal.events,
				splitShare{on: v.When(), security: v.Security, numerator: v.Numerator, denominator: v.Denominator},
			)
		default:
			return nil, fmt.Errorf("unhandled transaction type: %T", tx)
		}
	}
	return journal, nil
}
