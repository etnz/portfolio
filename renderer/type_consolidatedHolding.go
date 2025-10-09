package renderer

import "github.com/etnz/portfolio"

// ConsolidatedHolding represents a consolidated view of multiple holdings.
type ConsolidatedHolding struct {
	Date                            portfolio.Date  `json:"date"`
	ConsolidatedPortfolioValue      portfolio.Money `json:"consolidatedPortfolioValue"`
	ConsolidatedSecuritiesValue     portfolio.Money `json:"consolidatedSecuritiesValue"`
	ConsolidatedCashValue           portfolio.Money `json:"consolidatedCashValue"`
	ConsolidatedCounterpartiesValue portfolio.Money `json:"consolidatedCounterpartiesValue"`
	Holdings                        []*Holding      `json:"holdings"`
	ReportingCurrency               string          `json:"reportingCurrency"`
}

// NewConsolidatedHolding creates a new ConsolidatedHolding from a list of snapshots.
// It assumes the reporting currency of the first snapshot for consolidation.
func NewConsolidatedHolding(snapshots []*portfolio.Snapshot) *ConsolidatedHolding {
	if len(snapshots) == 0 {
		return &ConsolidatedHolding{}
	}

	// Use the first snapshot to determine the date and reporting currency for the whole consolidation.
	firstSnap := snapshots[0]
	on := firstSnap.On()
	reportingCurrency := firstSnap.ReportingCurrency()

	ch := &ConsolidatedHolding{
		Date:                            on,
		Holdings:                        make([]*Holding, 0, len(snapshots)),
		ReportingCurrency:               reportingCurrency,
		ConsolidatedPortfolioValue:      portfolio.M(0, reportingCurrency),
		ConsolidatedSecuritiesValue:     portfolio.M(0, reportingCurrency),
		ConsolidatedCashValue:           portfolio.M(0, reportingCurrency),
		ConsolidatedCounterpartiesValue: portfolio.M(0, reportingCurrency),
	}

	for _, s := range snapshots {
		h := NewHolding(s)
		ch.Holdings = append(ch.Holdings, h)

		// Convert and aggregate totals to the consolidated reporting currency.
		ch.ConsolidatedPortfolioValue = ch.ConsolidatedPortfolioValue.Add(firstSnap.Convert(h.TotalPortfolioValue))
		ch.ConsolidatedCashValue = ch.ConsolidatedCashValue.Add(firstSnap.Convert(h.TotalCashValue))
		ch.ConsolidatedSecuritiesValue = ch.ConsolidatedSecuritiesValue.Add(firstSnap.Convert(h.TotalSecuritiesValue))
		ch.ConsolidatedCounterpartiesValue = ch.ConsolidatedCounterpartiesValue.Add(firstSnap.Convert(h.TotalCounterpartiesValue))
	}

	return ch
}
