package portfolio

import "github.com/etnz/portfolio/date"

// HistoryReport represents a report on the history of an asset.
type HistoryReport struct {
	Security string
	Currency string
	Entries  []HistoryEntry
}

// HistoryEntry represents a single entry in the history report.
type HistoryEntry struct {
	Date     date.Date
	Position Quantity
	Price    Money
	Value    Money
}
