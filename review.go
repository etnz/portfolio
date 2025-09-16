package portfolio

import (
	"math"
)

// Review represents an analysis of the portfolio over a specific period (Range).
// It calculates period-based metrics by comparing two Snapshots: one at the
// start of the period and one at the end.
type Review struct {
	start *Snapshot // Snapshot at period.From - 1 day
	end   *Snapshot // Snapshot at period.To
}

// Start returns the snapshot at the beginning of the review period (taken on `period.From - 1`).
func (r *Review) Start() *Snapshot {
	return r.start
}

// End returns the snapshot at the end of the review period (taken on `period.To`).
func (r *Review) End() *Snapshot {
	return r.end
}

// Range returns the period range of the review.
func (r *Review) Range() Range {
	return NewRange(r.start.On().Add(1), r.end.On())
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

// PortfolioChange calculates the net change in total portfolio value during the review period.
func (r *Review) PortfolioChange() Money {
	return r.end.TotalPortfolio().Sub(r.start.TotalPortfolio())
}

// CashChange calculates the net change in the total cash balance during the review period.
func (r *Review) CashChange() Money {
	return r.end.TotalCash().Sub(r.start.TotalCash())
}

// CounterpartyChange calculates the net change in the total counterparty balance during the review period.
func (r *Review) CounterpartyChange() Money {
	return r.end.TotalCounterparty().Sub(r.start.TotalCounterparty())
}

// Transactions returns a slice of all transactions that occurred within the review period.
func (r *Review) Transactions() []Transaction {
	var periodTxs []Transaction
	periodRange := r.Range()
	// This assumes the journal is accessible via the snapshot.
	for _, e := range r.end.journal.events {
		if periodRange.Contains(e.date()) {
			// This is inefficient as it might add the same transaction multiple times
			// if it generated multiple events. We need a way to get the source transaction
			// and add it only once.
			// TODO: This needs a more efficient implementation, likely by iterating transactions directly.
			// For now, this provides a basic, albeit slow, implementation.
			tx := r.end.journal.transactionFromEvent(e)
			if len(periodTxs) == 0 || !periodTxs[len(periodTxs)-1].Equal(tx) {
				periodTxs = append(periodTxs, tx)
			}
		}
	}
	return periodTxs
}

// AssetNetTradingFlow calculates the net cash invested into or divested from a single security during the period.
func (r *Review) AssetNetTradingFlow(ticker string) (Money, error) {
	endFlow := r.end.NetTradingFlow(ticker)
	startFlow := r.start.NetTradingFlow(ticker)
	return endFlow.Sub(startFlow), nil
}

// AssetRealizedGains calculates the realized gains for a single security during the period.
func (r *Review) AssetRealizedGains(ticker string, method CostBasisMethod) (Money, error) {
	endGains := r.end.RealizedGains(ticker, method)
	startGains := r.start.RealizedGains(ticker, method)
	return endGains.Sub(startGains), nil
}

// AssetDividends calculates the dividends received for a single security during the period.
func (r *Review) AssetDividends(ticker string) (Money, error) {
	endDividends := r.end.Dividends(ticker)
	startDividends := r.start.Dividends(ticker)
	return endDividends.Sub(startDividends), nil
}

// AssetMarketGainLoss calculates the change in a security's value due to price movements during the period.
func (r *Review) AssetMarketGainLoss(ticker string) (Money, error) {
	valueChange := r.end.MarketValue(ticker).Sub(r.start.MarketValue(ticker))
	tradingFlow, err := r.AssetNetTradingFlow(ticker)
	if err != nil {
		return Money{}, err
	}
	return valueChange.Sub(tradingFlow), nil
}
