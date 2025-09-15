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
func NewHistory(ledger *Ledger, security, currency, reportingCurrency string) (*HistoryReport, error) {
	report := &HistoryReport{
		Security: security,
		Currency: currency,
		Entries:  []HistoryEntry{},
	}
	journal := ledger.journal
	if journal == nil {
		return report, nil
	}

	var predicate func(Transaction) bool
	if security != "" {
		predicate = BySecurity(security)
	} else {
		predicate = ledger.ByCurrency(currency)
	}

	// Build a list of all days where there was a significant transaction.
	days := make([]Date, 0, 100) // Pre-allocate
	previous := Date{}
	for _, tx := range ledger.Transactions(predicate) {
		on := tx.When()
		if on == previous {
			continue // already done for that day
		}
		previous = on
		days = append(days, on)
	}

	for _, on := range days {
		balance, err := NewBalance(journal, on, FIFO)
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
