package renderer

import "github.com/etnz/portfolio"

// ConsolidatedReview aggregates multiple reviews into a single consolidated report.
type ConsolidatedReview struct {
	AsOf              string          `json:"asOf"`
	Range             portfolio.Range `json:"range"`
	Reviews           []*Review       `json:"reviews"`
	ReportingCurrency string          `json:"reportingCurrency"`

	ConsolidatedTotalPortfolioValue      portfolio.Money `json:"consolidatedTotalPortfolioValue"`
	ConsolidatedTotalCashValue           portfolio.Money `json:"consolidatedTotalCashValue"`
	ConsolidatedTotalCounterpartiesValue portfolio.Money `json:"consolidatedTotalCounterpartiesValue"`
	ConsolidatedPreviousValue            portfolio.Money `json:"consolidatedPreviousValue"`
	ConsolidatedCapitalFlow              portfolio.Money `json:"consolidatedCapitalFlow"`
	ConsolidatedMarketGains              portfolio.Money `json:"consolidatedMarketGains"`
	ConsolidatedForexGains               portfolio.Money `json:"consolidatedForexGains"`
	ConsolidatedNetChange                portfolio.Money `json:"consolidatedNetChange"`
	ConsolidatedCashChange               portfolio.Money `json:"consolidatedCashChange"`
	ConsolidatedCounterpartiesChange     portfolio.Money `json:"consolidatedCounterpartiesChange"`
	ConsolidatedMarketValueChange        portfolio.Money `json:"consolidatedMarketValueChange"`
	ConsolidatedDividends                portfolio.Money `json:"consolidatedDividends"`
	ConsolidatedTotalGains               portfolio.Money `json:"consolidatedTotalGains"`
	ConsolidatedTotalStartMarketValue    portfolio.Money `json:"consolidatedTotalStartMarketValue"`
	ConsolidatedTotalEndMarketValue      portfolio.Money `json:"consolidatedTotalEndMarketValue"`
	ConsolidatedTotalNetTradingFlow      portfolio.Money `json:"consolidatedTotalNetTradingFlow"`
	ConsolidatedTotalRealizedGains       portfolio.Money `json:"consolidatedTotalRealizedGains"`
	ConsolidatedTotalUnrealizedGains     portfolio.Money `json:"consolidatedTotalUnrealizedGains"`
}

// NewConsolidatedReview creates a new ConsolidatedReview from a list of portfolio.Review objects.
func NewConsolidatedReview(portfolioReviews []*portfolio.Review, method portfolio.CostBasisMethod) *ConsolidatedReview {
	if len(portfolioReviews) == 0 {
		return &ConsolidatedReview{}
	}

	firstReview := portfolioReviews[0]
	reportingCurrency := firstReview.End().ReportingCurrency()

	cr := &ConsolidatedReview{
		AsOf:              Now().Format("2006-01-02 15:04:05"),
		Range:             firstReview.Range(),
		Reviews:           make([]*Review, 0, len(portfolioReviews)),
		ReportingCurrency: reportingCurrency,

		ConsolidatedTotalPortfolioValue:      portfolio.M(0, reportingCurrency),
		ConsolidatedTotalCashValue:           portfolio.M(0, reportingCurrency),
		ConsolidatedTotalCounterpartiesValue: portfolio.M(0, reportingCurrency),
		ConsolidatedPreviousValue:            portfolio.M(0, reportingCurrency),
		ConsolidatedCapitalFlow:              portfolio.M(0, reportingCurrency),
		ConsolidatedMarketGains:              portfolio.M(0, reportingCurrency),
		ConsolidatedForexGains:               portfolio.M(0, reportingCurrency),
		ConsolidatedNetChange:                portfolio.M(0, reportingCurrency),
		ConsolidatedCashChange:               portfolio.M(0, reportingCurrency),
		ConsolidatedCounterpartiesChange:     portfolio.M(0, reportingCurrency),
		ConsolidatedMarketValueChange:        portfolio.M(0, reportingCurrency),
		ConsolidatedDividends:                portfolio.M(0, reportingCurrency),
		ConsolidatedTotalGains:               portfolio.M(0, reportingCurrency),
		ConsolidatedTotalStartMarketValue:    portfolio.M(0, reportingCurrency),
		ConsolidatedTotalEndMarketValue:      portfolio.M(0, reportingCurrency),
		ConsolidatedTotalNetTradingFlow:      portfolio.M(0, reportingCurrency),
		ConsolidatedTotalRealizedGains:       portfolio.M(0, reportingCurrency),
		ConsolidatedTotalUnrealizedGains:     portfolio.M(0, reportingCurrency),
	}

	for _, pr := range portfolioReviews {
		r := NewReview(pr, method)
		cr.Reviews = append(cr.Reviews, r)

		cr.ConsolidatedTotalPortfolioValue = cr.ConsolidatedTotalPortfolioValue.Add(r.TotalPortfolioValue)
		cr.ConsolidatedTotalCashValue = cr.ConsolidatedTotalCashValue.Add(r.TotalCashValue)
		cr.ConsolidatedTotalCounterpartiesValue = cr.ConsolidatedTotalCounterpartiesValue.Add(r.TotalCounterpartiesValue)
		cr.ConsolidatedPreviousValue = cr.ConsolidatedPreviousValue.Add(r.PreviousValue)
		cr.ConsolidatedCapitalFlow = cr.ConsolidatedCapitalFlow.Add(r.CapitalFlow)
		cr.ConsolidatedMarketGains = cr.ConsolidatedMarketGains.Add(r.MarketGains)
		cr.ConsolidatedForexGains = cr.ConsolidatedForexGains.Add(r.ForexGains)
		cr.ConsolidatedNetChange = cr.ConsolidatedNetChange.Add(r.NetChange)
		cr.ConsolidatedCashChange = cr.ConsolidatedCashChange.Add(r.CashChange)
		cr.ConsolidatedCounterpartiesChange = cr.ConsolidatedCounterpartiesChange.Add(r.CounterpartiesChange)
		cr.ConsolidatedMarketValueChange = cr.ConsolidatedMarketValueChange.Add(r.MarketValueChange)
		cr.ConsolidatedDividends = cr.ConsolidatedDividends.Add(r.Dividends)
		cr.ConsolidatedTotalGains = cr.ConsolidatedTotalGains.Add(r.TotalGains)
		cr.ConsolidatedTotalStartMarketValue = cr.ConsolidatedTotalStartMarketValue.Add(r.TotalStartMarketValue)
		cr.ConsolidatedTotalEndMarketValue = cr.ConsolidatedTotalEndMarketValue.Add(r.TotalEndMarketValue)
		cr.ConsolidatedTotalNetTradingFlow = cr.ConsolidatedTotalNetTradingFlow.Add(r.TotalNetTradingFlow)
		cr.ConsolidatedTotalRealizedGains = cr.ConsolidatedTotalRealizedGains.Add(r.TotalRealizedGains)
		cr.ConsolidatedTotalUnrealizedGains = cr.ConsolidatedTotalUnrealizedGains.Add(r.TotalUnrealizedGains)
	}

	return cr
}
