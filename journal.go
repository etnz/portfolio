package portfolio

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// event represents a single, atomic operation in the portfolio's history.
// It is the lowest-level, immutable fact from which all states are derived.
type event interface {
	date() Date
	source() int // index of the transaction that created that event
}

// Journal holds a chronologically sorted list of all atomic events.
type Journal struct {
	cur    string  // the reporting currency.
	events []event // sorted by date
	txs    []Transaction
}

type baseEvent struct {
	on  Date
	src int
}

func (e baseEvent) date() Date  { return e.on }
func (e baseEvent) source() int { return e.src }

// --- Cash Events ---

// creditCash increases the balance of a cash account.
type creditCash struct {
	baseEvent
	amount   Money
	external bool // true when cash comes from outside.
}

func (e creditCash) currency() string { return e.amount.Currency() }

// debitCash decreases the balance of a cash account.
type debitCash struct {
	baseEvent
	amount   Money
	external bool // true when cash goes outside.
}

func (e debitCash) currency() string { return e.amount.Currency() }

// --- Security Events ---

// acquireLot adds a new lot of a security.
type acquireLot struct {
	baseEvent
	security string
	quantity Quantity
	cost     Money
}

// disposeLot removes a quantity of a security.
type disposeLot struct {
	baseEvent
	security string
	quantity Quantity
	proceeds Money
}

// receiveDividend logs the receipt of a dividend payment.
// This is treated as income to the owner, not a cash flow into the portfolio.
type receiveDividend struct {
	baseEvent
	security string
	amount   Money // per share.
}

// --- Counterparty Events ---

// declareCounterparty maps a ticker to a security ID and currency.
type declareCounterparty struct {
	baseEvent
	account  string
	currency string
}

// creditCounterparty increases an asset (receivable) or reduces a liability (payable).
type creditCounterparty struct {
	baseEvent
	account  string
	amount   Money
	external bool // true when money goes outside.

}

func (e creditCounterparty) currency() string { return e.amount.Currency() }

// debitCounterparty decreases an asset (receivable) or increases a liability (payable).
type debitCounterparty struct {
	baseEvent
	account  string
	amount   Money
	external bool // true when money goes outside.
}

func (e debitCounterparty) currency() string { return e.amount.Currency() }

// --- Market and Metadata Events ---

// splitShare adjusts the quantity of existing lots for a security.
type splitShare struct {
	baseEvent
	security    string
	numerator   int64
	denominator int64
}

// declareSecurity maps a ticker to a security ID and currency.
type declareSecurity struct {
	baseEvent
	ticker   string
	id       ID
	currency string
}

// updatePrice sets the price of a security on a given date.
type updatePrice struct {
	baseEvent
	security string
	price    Money
}

// updateForex sets the price of a security on a given date.
type updateForex struct {
	baseEvent
	currency string // the foreign currency (USD in USDEUR)
	rate     Money  //(the cost of 1 USD in EUR (for USDEUR forex))
}

// NewJournal converts a Ledger of high-level transactions and market data events
// into a Journal of low-level, atomic events.
func (ledger *Ledger) newJournal() error {
	ledger.stableSort()
	journal := &Journal{
		events: make([]event, 0, len(ledger.transactions)*2), // Pre-allocate
		txs:    ledger.transactions,
		cur:    ledger.currency,
	}

	for src, tx := range ledger.transactions {
		b := baseEvent{on: tx.When(), src: src}
		switch v := tx.(type) {
		case Buy:
			sec := ledger.Security(v.Security)
			if sec == nil {
				return fmt.Errorf("security %q not declared for buy transaction on %s", v.Security, v.When())
			}

			journal.events = append(journal.events,
				acquireLot{baseEvent: b, security: v.Security, quantity: v.Quantity, cost: v.Amount},
				debitCash{baseEvent: b, amount: v.Amount, external: false},
			)
		case Sell:
			sec := ledger.Security(v.Security)
			if sec == nil {
				return fmt.Errorf("security %q not declared for sell transaction on %s", v.Security, v.When())
			}
			journal.events = append(journal.events,
				disposeLot{baseEvent: b, security: v.Security, quantity: v.Quantity, proceeds: v.Amount},
				creditCash{baseEvent: b, amount: v.Amount, external: false},
			)
		case Dividend:
			sec := ledger.Security(v.Security)
			if sec == nil {
				return fmt.Errorf("security %q not declared for dividend transaction on %s", v.Security, v.When())
			}
			journal.events = append(journal.events,
				receiveDividend{baseEvent: b, security: v.Security, amount: v.Amount},
			)
		case Deposit:
			amount := v.Amount
			// A deposit that settles a receivable is not considered as external (since the amount)
			// was taken into account when accruing the receivable
			ext := v.Settles == ""
			journal.events = append(journal.events,
				creditCash{baseEvent: b, amount: amount, external: ext},
			)
			if v.Settles != "" {
				// A deposit settling an account means a counterparty paid us back, reducing what they owe us (asset).
				journal.events = append(journal.events,
					debitCounterparty{baseEvent: b, account: v.Settles, amount: amount},
				)
			}
		case Withdraw:
			amount := v.Amount
			// A withdrawal that settles a payable is not considered as external (since the amount)
			// was taken into account when accruing the receivable
			ext := v.Settles == ""
			journal.events = append(journal.events,
				debitCash{baseEvent: b, amount: amount, external: ext},
			)
			if v.Settles != "" {
				// A withdrawal settling an account means we paid a counterparty back, reducing what we owe them (liability).
				journal.events = append(journal.events,
					creditCounterparty{baseEvent: b, account: v.Settles, amount: amount},
				)
			}
		case Convert:
			journal.events = append(journal.events,
				debitCash{baseEvent: b, amount: v.FromAmount},
				creditCash{baseEvent: b, amount: v.ToAmount},
			)
		case Declare:
			if _, _, err := v.ID.CurrencyPair(); err == nil {
				continue // do not declare currency as securities
			}

			journal.events = append(journal.events,
				declareSecurity{baseEvent: b, ticker: v.Ticker, id: v.ID, currency: v.Currency},
			)
		case Accrue:
			if v.Create {
				journal.events = append(journal.events, declareCounterparty{baseEvent: b, account: v.Counterparty, currency: v.Currency()})
			}
			amount := v.Amount
			if amount.IsPositive() { // Receivable: counterparty owes us (asset) -> increase asset
				journal.events = append(journal.events,
					creditCounterparty{baseEvent: b, account: v.Counterparty, amount: amount, external: true},
				)
			} else { // Payable: we owe counterparty (liability) -> increase liability
				journal.events = append(journal.events,
					debitCounterparty{baseEvent: b, account: v.Counterparty, amount: amount.Neg(), external: true},
				)
			}
		case UpdatePrice:
			for ticker, priceDecimal := range v.PricesIter() {
				sec := ledger.Security(ticker)
				if sec == nil {
					// This should have been caught by validation, but we check again.
					return fmt.Errorf("security %q from update-price not declared", ticker)
				}
				price := M(priceDecimal, sec.Currency())

				// Handle forex updates
				if base, quote, err := sec.ID().CurrencyPair(); err == nil {
					if quote == journal.cur {
						journal.events = append(journal.events,
							updateForex{baseEvent: b, currency: base, rate: price},
						)
					}
					if base == journal.cur {
						p := M(decimal.NewFromInt(1).Div(price.value), base)
						p.value = p.value.Round(5) // is enought for an approximate price anyway.
						journal.events = append(journal.events,
							updateForex{baseEvent: b, currency: quote, rate: p},
						)
					}
					continue
				}

				// Handle regular security price updates
				journal.events = append(journal.events,
					updatePrice{baseEvent: b, security: ticker, price: price},
				)
			}

		case Split:
			journal.events = append(journal.events,
				splitShare{baseEvent: b, security: v.Security, numerator: v.Numerator, denominator: v.Denominator},
			)
		default:
			return fmt.Errorf("unhandled transaction type: %T", tx)
		}
	}
	ledger.journal = journal
	return nil
}

// CashBalance computes the total cash in a specific currency on a specific date.
func (j *Journal) CashBalance(on Date, currency string) Money {
	balance := M(decimal.Zero, currency)
	for _, e := range j.events {
		if e.date().After(on) {
			break
		}
		switch v := e.(type) {
		case creditCash:
			if v.currency() == currency {
				balance = balance.Add(v.amount)
			}
		case debitCash:
			if v.currency() == currency {
				balance = balance.Sub(v.amount)
			}
		}
	}
	return balance
}

func (j *Journal) transactionFromEvent(e event) Transaction {
	return j.txs[e.source()]
}
