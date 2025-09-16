package portfolio

import (
	"fmt"
	"math"
)

// Review represents an analysis of the portfolio over a specific period (Range).
// It calculates period-based metrics by comparing two Snapshots: one at the
// start of the period and one at the end.
type Review struct {
	start *Snapshot // Snapshot at period.From - 1 day
	end   *Snapshot // Snapshot at period.To
}

// NewReview creates a new portfolio review for a given period.
// It initializes the start and end snapshots needed for period calculations.
func NewReview(journal *Journal, period Range) (*Review, error) {
	if journal == nil {
		return nil, fmt.Errorf("cannot create review with a nil journal")
	}

	startSnapshot, err := NewSnapshot(journal, period.From.Add(-1))
	if err != nil {
		return nil, fmt.Errorf("failed to create start snapshot: %w", err)
	}

	endSnapshot, err := NewSnapshot(journal, period.To)
	if err != nil {
		return nil, fmt.Errorf("failed to create end snapshot: %w", err)
	}

	r := &Review{
		start: startSnapshot,
		end:   endSnapshot,
	}
	return r, nil
}

// CashFlow calculates the total net cash that has moved into or out of the
// portfolio from external sources during the review period.
func (r *Review) CashFlow() Money {
	return r.end.TotalCashFlow().Sub(r.start.TotalCashFlow())
}

// NetTradingFlow calculates the total net cash invested into or divested from
// all securities during the review period.
func (r *Review) NetTradingFlow() Money {
	return r.end.TotalNetTradingFlow().Sub(r.start.TotalNetTradingFlow())
}

// RealizedGains calculates the sum of all profits and losses 'locked in'
// through the sale of securities during the review period.
func (r *Review) RealizedGains(method CostBasisMethod) Money {
	return r.end.TotalRealizedGains(method).Sub(r.start.TotalRealizedGains(method))
}

// Dividends calculates the total income received from dividends
// during the review period.
func (r *Review) Dividends() Money {
	return r.end.TotalDividends().Sub(r.start.TotalDividends())
}

// TimeWeightedReturn calculates the compound rate of growth for a security
// over the review period, eliminating the distorting effects of cash flows.
func (r *Review) TimeWeightedReturn(ticker string) Percent {
	startVAV := r.start.VirtualAssetValue(ticker)
	endVAV := r.end.VirtualAssetValue(ticker)
	if startVAV.IsZero() {
		return Percent(math.NaN())
	}
	return Percent(100 * (endVAV.AsFloat()/startVAV.AsFloat() - 1))
}

// MarketGainLoss calculates the change in security value due to price movements,
// isolated from the impact of buying or selling.
func (r *Review) MarketGainLoss() Money {
	tmvChange := r.end.TotalMarket().Sub(r.start.TotalMarket())
	netTradingFlow := r.NetTradingFlow()
	return tmvChange.Sub(netTradingFlow)
}

// TotalReturn calculates the total economic benefit from the portfolio over a period,
// combining market gains/losses and dividend income.
func (r *Review) TotalReturn() Money {
	marketGainLoss := r.MarketGainLoss()
	dividends := r.Dividends()
	return marketGainLoss.Add(dividends)
}
