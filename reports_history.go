package portfolio

import (
	"fmt"
)

// HistoryReport represents a report on the history of an asset.
type HistoryReport struct {
	Security string
	Currency string
	Entries  []HistoryEntry
}

// HistoryEntry represents a single entry in the history report.
type HistoryEntry struct {
	Date     Date
	Position Quantity
	Price    Money
	Value    Money
}

// NewHistory computes the history of a security or currency.
func (as *AccountingSystem) NewHistory(security, currency string) (*HistoryReport, error) {
	// TODO: this is not a report, so it should not be here
	report := &HistoryReport{
		Security: security,
		Currency: currency,
		Entries:  []HistoryEntry{},
	}

	var predicate func(Transaction) bool
	if security != "" {
		predicate = BySecurity(security)
	} else {
		predicate = as.Ledger.ByCurrency(currency)
	}

	// Build a list of all days where there was a significant transaction.
	days := make([]Date, 0, len(as.Ledger.transactions))
	previous := Date{}
	for _, tx := range as.Ledger.Transactions(predicate) {
		on := tx.When()
		if on == previous {
			continue // already done for that day
		}
		previous = on
		days = append(days, on)
	}

	for _, on := range days {
		balance, err := as.Balance(on)
		if err != nil {
			return nil, fmt.Errorf("could not get balance for %s: %w", on, err)
		}

		var entry HistoryEntry
		if security != "" {
			entry = HistoryEntry{
				Date:     on,
				Position: balance.Position(security),
				Price:    balance.Price(security),
				Value:    balance.MarketValue(security),
			}
		} else {
			entry = HistoryEntry{
				Date:  on,
				Value: balance.Cash(currency),
			}
		}
		report.Entries = append(report.Entries, entry)
	}

	return report, nil
}
