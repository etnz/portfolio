package portfolio

import (
	"fmt"
	"time"
)

// ReviewReport contains the transactions for a given period.
type ReviewReport struct {
	// Range of the report all days included in the report.
	Range Range
	// Timestamp is the timestamp of the report generation.
	Timestamp time.Time
	// Reporting Currency
	ReportingCurrency string

	PortfolioValue Performance
	Cash           Performance // Variation of total cash in accounts.
	Counterparty   Performance // Variation of Counterpary Value.
	CashFlow       Money       // Algebraic sum of money crossing the boundaries of the portfolio (in/out)
	// Gains              *GainsReport
	CashAccounts   []CashAccountReview
	Counterparties []CoutnerpartyAccountReview
	Transactions   []Transaction
	Assets         []AssetReview
	Total          AssetReview // Total of asset review columns.
}

// AssetReview provides a summary of an asset's performance over a period.
type AssetReview struct {
	Security         string
	StartingPosition Quantity
	EndingPosition   Quantity
	Value            Performance
	Buys             Money
	Sells            Money
	Dividends        Money
	RealizedGains    Money
	UnrealizedGains  Money
}

// Flow returns the net trading flow for the asset (Sells - Buys).
// A positive value indicates more was sold than bought, representing a net cash inflow from this asset's trading activity.
func (ar AssetReview) Flow() Money {
	return ar.Buys.Sub(ar.Sells)
}

// Gain returns the net gain from holding the asset, excluding trading flow.
// It's calculated as the change in market value minus the net trading flow.
func (ar AssetReview) Gain() Money {
	return ar.Value.Change().Sub(ar.Flow())
}

// TotalReturn returns the total economic benefit from the asset, including market gains and dividends.
func (ar AssetReview) TotalReturn() Money {
	// Note: ar.Gain() already includes realized and unrealized gains.
	return ar.Gain().Add(ar.Dividends)
}

type CashAccountReview struct {
	Label  string
	Value  Money
	Return Percent
}
type CoutnerpartyAccountReview struct {
	Label string
	Value Money
}

// NewReviewReport returns a report with all transactions in a given period.
func NewReviewReport(ledger *Ledger, reportingCurrency string, period Range) (*ReviewReport, error) {
	journal, err := newJournal(ledger, reportingCurrency)
	if err != nil {
		return nil, err
	}

	// Compute the balance on the last days and on the day before the first to compute several
	// different metrics from there.
	endBalance, err := NewBalance(journal, period.To, AverageCost)
	if err != nil {
		return nil, fmt.Errorf("failed to compute end balance on %s: %w", period.To, err)
	}

	// compute the balance the day before the first day of the period.
	startBalance, err := NewBalance(journal, period.From.Add(-1), AverageCost)
	if err != nil {
		return nil, err
	}

	// Calculate the cash flow over the period per currency
	// and summing them up (in the reporting currency)
	// Computing the $total end cash flow - total start cash flow cause$
	// is counter intuitive as it shows also the gains made playing with forex.
	var totalCashFlow Money
	for cur := range endBalance.Currencies() {
		flow := endBalance.CashFlow(cur).Sub(startBalance.CashFlow(cur))
		totalCashFlow = totalCashFlow.Add(endBalance.Convert(flow))
	}

	// Same for Counterparties
	var totalCounterparties Money
	for acc := range endBalance.Counterparties() {
		change := endBalance.Counterparty(acc).Sub(startBalance.Counterparty(acc))
		totalCounterparties = totalCounterparties.Add(endBalance.Convert(change))
	}

	cashAccounts := make([]CashAccountReview, 0, 100)
	for cur := range endBalance.Currencies() {
		var forexReturn Percent
		startRate := startBalance.forex[cur]
		endRate := endBalance.forex[cur]
		if !startRate.IsZero() {
			forexReturn = Percent(100 * (endRate.AsFloat() - startRate.AsFloat()) / startRate.AsFloat())
		}
		cashAccounts = append(cashAccounts, CashAccountReview{
			Label:  cur,
			Value:  endBalance.Cash(cur),
			Return: forexReturn,
		})
	}

	counterpartyAccounts := make([]CoutnerpartyAccountReview, 0, 100)
	for acc := range endBalance.Counterparties() {
		counterpartyAccounts = append(counterpartyAccounts, CoutnerpartyAccountReview{
			Label: acc,
			Value: endBalance.Counterparty(acc),
		})
	}

	// Create the transaction in this range.
	transactions := make([]Transaction, 0, 1000)
	for _, tx := range ledger.transactions {
		if period.Contains(tx.When()) {
			transactions = append(transactions, tx)
		}
	}
	// Sum realized gains in the period per security
	total := AssetReview{
		Security: "Total",
		Value:    NewPerformance(startBalance.TotalMarketValue(), endBalance.TotalMarketValue()),
	}

	assets := make([]AssetReview, 0, 100)

	// Calculate Security breakdown
	for s := range endBalance.Securities() {
		ticker := s.Ticker()
		startPos := startBalance.Position(ticker)
		startValue := startBalance.MarketValue(ticker)
		endPos := endBalance.Position(ticker)
		endValue := endBalance.MarketValue(ticker)

		// Calculate flows and gains within the period
		buysInPeriod := endBalance.Buys(ticker).Sub(startBalance.Buys(ticker))
		sellsInPeriod := endBalance.Sells(ticker).Sub(startBalance.Sells(ticker))
		dividendsInPeriod := endBalance.DividendsReceived(ticker).Sub(startBalance.DividendsReceived(ticker))
		realizedGain := endBalance.RealizedGain(ticker).Sub(startBalance.RealizedGain(ticker))
		unrealizedGain := endBalance.MarketValue(ticker).Sub(endBalance.CostBasis(ticker))

		// Sum up total realized gains for the report summary
		total.Buys = total.Buys.Add(endBalance.Convert(buysInPeriod))
		total.Sells = total.Sells.Add(endBalance.Convert(sellsInPeriod))
		total.Dividends = total.Dividends.Add(endBalance.Convert(dividendsInPeriod))
		total.RealizedGains = total.RealizedGains.Add(endBalance.Convert(realizedGain))
		total.UnrealizedGains = total.UnrealizedGains.Add(endBalance.Convert(unrealizedGain))

		// Skip assets that were not held and had no activity during the period
		if startPos.IsZero() && endPos.IsZero() && realizedGain.IsZero() && buysInPeriod.IsZero() && sellsInPeriod.IsZero() && unrealizedGain.IsZero() {
			continue
		}

		// Calculate the price return for the period
		startPrice := startBalance.Price(ticker)
		endPrice := endBalance.Price(ticker)
		priceReturn := Percent(100 * (endPrice.AsFloat() - startPrice.AsFloat()) / startPrice.AsFloat())
		// Fill the AssetReview with the Data.
		assets = append(assets, AssetReview{
			Security:         ticker,
			StartingPosition: startPos,
			EndingPosition:   endPos,
			Value:            NewPerformanceWithReturn(startValue, endValue, priceReturn),
			Buys:             buysInPeriod,
			Sells:            sellsInPeriod,
			Dividends:        dividendsInPeriod,
			RealizedGains:    realizedGain,
			UnrealizedGains:  unrealizedGain,
		})
	}

	totalTWR := Percent((endBalance.linkedTWR/startBalance.linkedTWR - 1) * 100)
	// Calculate top metrics.
	report := &ReviewReport{
		Range:             period, // TODO: check if this is correct
		ReportingCurrency: reportingCurrency,
		PortfolioValue:    NewPerformanceWithReturn(startBalance.TotalPortfolioValue(), endBalance.TotalPortfolioValue(), totalTWR),
		Cash:              NewPerformance(startBalance.TotalCash(), endBalance.TotalCash()),
		Counterparty:      NewPerformance(startBalance.TotalCounterparty(), endBalance.TotalCounterparty()),
		CashFlow:          totalCashFlow,
		CashAccounts:      cashAccounts,
		Counterparties:    counterpartyAccounts,
		Assets:            assets,
		Transactions:      transactions,
		Total:             total,
	}

	return report, nil
}

func (r *ReviewReport) NetGains() Money {
	return r.PortfolioValue.End.Sub(r.PortfolioValue.Start).Sub(r.CashFlow)
}

// TotalReturn returns the total economic benefit from the portfolio, including market gains and dividends.
func (r *ReviewReport) TotalReturn() Money {
	return r.NetGains().Add(r.Total.Dividends)
}
