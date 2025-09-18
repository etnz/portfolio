package portfolio

import (
	"maps"
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
	periodRange := r.Range()
	flowsByCurrency := make(map[string]Money)

	for _, e := range r.end.journal.events {
		eventDate := e.date()
		if eventDate.Before(periodRange.From) {
			continue
		}
		if eventDate.After(periodRange.To) {
			break
		}

		switch v := e.(type) {
		case creditCash:
			if v.external {
				cur := v.currency()
				flowsByCurrency[cur] = flowsByCurrency[cur].Add(v.amount)
			}
		case debitCash:
			if v.external {
				cur := v.currency()
				flowsByCurrency[cur] = flowsByCurrency[cur].Sub(v.amount)
			}
		case creditCounterparty:
			if v.external {
				cur := v.currency()
				flowsByCurrency[cur] = flowsByCurrency[cur].Add(v.amount)
			}
		case debitCounterparty:
			if v.external {
				cur := v.currency()
				flowsByCurrency[cur] = flowsByCurrency[cur].Sub(v.amount)
			}
		}
	}

	return r.end.sum(maps.Keys(flowsByCurrency), func(cur string) Money { return flowsByCurrency[cur] })
}

// NetTradingFlow calculates the total net cash invested into or divested from
// all securities during the review period.
func (r *Review) NetTradingFlow() Money {
	total := M(0, r.end.journal.cur)
	for ticker := range r.end.Securities() {
		flow := r.AssetNetTradingFlow(ticker)
		total = total.Add(r.end.Convert(flow))
	}
	return total
}

// RealizedGains calculates the sum of all profits and losses 'locked in'
// through the sale of securities during the review period.
func (r *Review) RealizedGains(method CostBasisMethod) Money {
	total := M(0, r.end.journal.cur)
	for ticker := range r.end.Securities() {
		gain := r.AssetRealizedGains(ticker, method)
		total = total.Add(r.end.Convert(gain))
	}
	return total
}

// Dividends calculates the total income received from dividends
// during the review period.
func (r *Review) Dividends() Money {
	total := M(0, r.end.journal.cur)
	for ticker := range r.end.Securities() {
		dividend := r.AssetDividends(ticker)
		total = total.Add(r.end.Convert(dividend))
	}
	return total
}

// TimeWeightedReturn calculates the compound rate of growth for a security
// over the review period, eliminating the distorting effects of cash flows.
func (r *Review) TimeWeightedReturn() Percent {

	den := r.end.TotalPortfolio().Sub(r.CashFlow())
	num := r.start.TotalPortfolio()
	if num.IsZero() {
		return Percent(math.NaN())
	}
	return Percent(100 * (den.AsFloat()/num.AsFloat() - 1))
}

// MarketGain calculates the change in security value due to price movements,
// isolated from the impact of buying or selling.
func (r *Review) MarketGain() Money {
	total := M(0, r.end.journal.cur)
	for ticker := range r.end.Securities() {
		gain := r.AssetMarketGain(ticker)
		total = total.Add(r.end.Convert(gain))
	}
	return total
}

// TotalReturn calculates the total economic benefit from the portfolio over a period,
// combining market gains/losses and dividend income.
func (r *Review) TotalReturn() Money {
	marketGainLoss := r.MarketGain()
	dividends := r.Dividends()
	return marketGainLoss.Add(dividends)
}

// DividendReturn calculates the return from dividends as a percentage of the starting portfolio value.
func (r *Review) DividendReturn() Percent {
	dividends := r.Dividends()
	startValue := r.start.TotalPortfolio()
	if startValue.IsZero() {
		return Percent(0)
	}
	return Percent(100 * dividends.AsFloat() / startValue.AsFloat())
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

// TotalMarketChange calculates the net change in the total market value of all securities during the review period.
func (r *Review) TotalMarketChange() Money {
	return r.end.TotalMarket().Sub(r.start.TotalMarket())
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

// AssetTimeWeightedReturn calculates the time-weighted return for a single security over the review period.
func (r *Review) AssetTimeWeightedReturn(ticker string) Percent {
	startValue := r.start.VirtualAssetValue(ticker)
	endValue := r.end.VirtualAssetValue(ticker)
	if startValue.IsZero() {
		return Percent(math.NaN())
	}
	return Percent(100 * (endValue.AsFloat()/startValue.AsFloat() - 1))
}

// CurrencyTimeWeightedReturn calculates the time-weighted return for a currency's exchange rate over the review period.
func (r *Review) CurrencyTimeWeightedReturn(currency string) Percent {
	startRate := r.start.ExchangeRate(currency)
	endRate := r.end.ExchangeRate(currency)
	if startRate.IsZero() {
		return Percent(math.NaN())
	}
	// rate is value of 1 unit of foreign currency in reporting currency.
	return Percent(100 * (endRate.AsFloat()/startRate.AsFloat() - 1))
}

// AssetNetTradingFlow calculates the net cash invested into or divested from a single security during the period.
func (r *Review) AssetNetTradingFlow(ticker string) Money {
	endFlow := r.end.NetTradingFlow(ticker)
	startFlow := r.start.NetTradingFlow(ticker)
	return endFlow.Sub(startFlow)
}

// AssetRealizedGains calculates the realized gains for a single security during the period.
func (r *Review) AssetRealizedGains(ticker string, method CostBasisMethod) Money {
	endGains := r.end.RealizedGains(ticker, method)
	startGains := r.start.RealizedGains(ticker, method)
	return endGains.Sub(startGains)
}

// AssetDividends calculates the dividends received for a single security during the period.
func (r *Review) AssetDividends(ticker string) Money {
	endDividends := r.end.Dividends(ticker)
	startDividends := r.start.Dividends(ticker)
	return endDividends.Sub(startDividends)
}

// AssetMarketGain calculates the change in a security's value due to price movements during the period.
func (r *Review) AssetMarketGain(ticker string) Money {
	valueChange := r.end.MarketValue(ticker).Sub(r.start.MarketValue(ticker))
	tradingFlow := r.AssetNetTradingFlow(ticker)
	return valueChange.Sub(tradingFlow)
}

// AssetTotalReturn calculates the total return for a single security during the period,
// combining market gains/losses and dividend income.
func (r *Review) AssetTotalReturn(ticker string) Money {
	marketGain := r.AssetMarketGain(ticker)
	dividends := r.AssetDividends(ticker)
	return marketGain.Add(dividends)
}

// AssetDividendReturn calculates the return from dividends for a single asset as a percentage of its starting value.
// If the asset was not held at the start of the period, the return is considered not applicable (NaN).
func (r *Review) AssetDividendReturn(ticker string) Percent {
	dividends := r.AssetDividends(ticker)
	startValue := r.start.MarketValue(ticker)

	if startValue.IsZero() {
		return Percent(math.NaN()) // Return is not applicable if starting value is zero.
	}
	return Percent(100 * r.end.Convert(dividends).AsFloat() / r.end.Convert(startValue).AsFloat())
}

// UnrealizedGains calculates the change in unrealized gains for a single security during the period.
func (r *Review) UnrealizedGains(method CostBasisMethod) Money {
	total := M(0, r.end.journal.cur)
	for ticker := range r.end.Securities() {
		gain := r.end.UnrealizedGains(ticker, method).Sub(r.start.UnrealizedGains(ticker, method))
		total = total.Add(r.end.Convert(gain))
	}
	return total
}

// AssetCostBasis calculates the cost basis of a single security at the end of the review period.
// This is used for the "Invested" column in reports.
func (r *Review) AssetCostBasis(ticker string, method CostBasisMethod) Money {
	return r.end.CostBasis(ticker, method)
}

// TotalCostBasis calculates the total cost basis of all securities held at the end of the review period.
// This is used for the "Invested" total in reports.
func (r *Review) TotalCostBasis(method CostBasisMethod) Money {
	total := M(0, r.end.journal.cur)
	for ticker := range r.end.Securities() {
		cost := r.AssetCostBasis(ticker, method)
		total = total.Add(r.end.Convert(cost))
	}
	return total
}
