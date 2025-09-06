package portfolio

import (
	"time"

	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

// ReviewReport contains the transactions for a given period.
type ReviewReport struct {
	// Range of the report all days included in the report.
	Range date.Range
	// Timestamp is the timestamp of the report generation.
	Timestamp time.Time
	// Reporting Currency
	ReportingCurrency string

	PortfolioValue Performance
	MarketValue    Performance
	Cash           Performance // Variation of total cash in accounts.
	Counterparty   Performance // Variation of Counterpary Value.

	CashFlow   Money // Algebraic sum of moeny crossing the boundaries of the portfolio (in/out)
	Unrealized Money // Total Unrealized gains at the end of the period.
	Realized   Money // Realized gains during the period.

	// Gains              *GainsReport
	CashAccounts   []CashAccountReview
	Counterparties []CoutnerpartyAccountReview
	Assets         []AssetReview
	Transactions   []Transaction
}

// AssetReview provides a summary of an asset's performance over a period.
type AssetReview struct {
	Security         string
	StartingPosition Quantity
	StartingValue    Money
	EndingPosition   Quantity
	EndingValue      Money
	RealizedGains    Money
	UnrealizedGains  Money
}

type CashAccountReview struct {
	Label string
	Value Money
}
type CoutnerpartyAccountReview struct {
	Label string
	Value Money
}

// NewReviewReport returns a report with all transactions in a given period.
func (as *AccountingSystem) NewReviewReport(period date.Range) (*ReviewReport, error) {

	// Compute the balance on the last days and on the day before the first to compute several
	// different metrics from there.
	endBalance, err := as.Balance(period.To)
	if err != nil {
		return nil, err
	}

	// compute the balance the day before the first day of the period.
	startBalance, err := as.Balance(period.From.Add(-1))
	if err != nil {
		return nil, err
	}

	// Calculate the cash flow over the period per currency
	// and summing them up (in the reporting currency)
	// Computing the $total end cash flow - total start cash flow cause$
	// is counter intuitive as it shows also the gains made playing with forex.
	totalCashFlow := decimal.Zero
	for cur := range endBalance.Currencies() {
		flow := endBalance.CashFlow(cur).Sub(startBalance.CashFlow(cur))
		totalCashFlow = totalCashFlow.Add(endBalance.Convert(flow, cur))
	}

	// Same for Counterparties
	totalCounterparties := decimal.Zero
	for acc := range endBalance.Counterparties() {
		cur := endBalance.CounterpartyCurrency(acc)
		change := endBalance.Counterparty(acc).Sub(startBalance.Counterparty(acc))
		totalCounterparties = totalCounterparties.Add(endBalance.Convert(change, cur))
	}

	// Sum realized gains in the period per security
	totalRealized := decimal.Zero

	cashAccounts := make([]CashAccountReview, 0, 100)
	for cur := range endBalance.Currencies() {
		cashAccounts = append(cashAccounts, CashAccountReview{
			Label: cur,
			Value: NewMoney(endBalance.Cash(cur), cur),
		})
	}

	counterpartyAccounts := make([]CoutnerpartyAccountReview, 0, 100)
	for acc := range endBalance.Counterparties() {
		cur := endBalance.CounterpartyCurrency(acc)
		counterpartyAccounts = append(counterpartyAccounts, CoutnerpartyAccountReview{
			Label: acc,
			Value: NewMoney(endBalance.Counterparty(acc), cur),
		})
	}

	// Calculate Security breakdown
	assets := make([]AssetReview, 0, len(endBalance.securities))

	for sec := range endBalance.Securities() {
		ticker := sec.Ticker()
		currency := sec.Currency()

		startPos := startBalance.Position(ticker)
		startValue := startBalance.MarketValue(ticker)
		endPos := endBalance.Position(ticker)
		endValue := endBalance.MarketValue(ticker)

		// Gains
		realizedGain := endBalance.RealizedGain(ticker).Sub(startBalance.RealizedGain(ticker))
		unrealizedGain := endBalance.MarketValue(ticker).Sub(endBalance.CostBasis(ticker))
		totalRealized = totalRealized.Add(endBalance.Convert(realizedGain, sec.currency))

		if startPos.IsZero() && endPos.IsZero() && realizedGain.IsZero() {
			continue
		}
		assets = append(assets, AssetReview{
			Security:         ticker,
			StartingPosition: NewQuantity(startPos),
			EndingPosition:   NewQuantity(endPos),
			StartingValue:    NewMoney(startValue, currency),
			EndingValue:      NewMoney(endValue, currency),
			RealizedGains:    NewMoney(realizedGain, currency),
			UnrealizedGains:  NewMoney(unrealizedGain, currency),
		})
	}

	// Create the transaction in this range.
	transactions := make([]Transaction, 0, 1000)
	for _, tx := range as.Ledger.transactions {
		if period.Contains(tx.When()) {
			transactions = append(transactions, tx)
		}
	}

	// Calculate top metrics.
	report := &ReviewReport{
		Range:             period,
		ReportingCurrency: as.ReportingCurrency,
		PortfolioValue:    NewPerformanceFromDecimal(startBalance.TotalPortfolioValue(), endBalance.TotalPortfolioValue(), as.ReportingCurrency),
		MarketValue:       NewPerformanceFromDecimal(startBalance.TotalMarketValue(), endBalance.TotalMarketValue(), as.ReportingCurrency),
		Cash:              NewPerformanceFromDecimal(startBalance.TotalCash(), endBalance.TotalCash(), as.ReportingCurrency),
		Counterparty:      NewPerformanceFromDecimal(startBalance.TotalCounterparty(), endBalance.TotalCounterparty(), as.ReportingCurrency),
		CashFlow:          NewMoney(totalCashFlow, as.ReportingCurrency),
		Realized:          NewMoney(totalRealized, as.ReportingCurrency),
		Unrealized:        NewMoney(endBalance.TotalUnrealizedGain(), as.ReportingCurrency),
		CashAccounts:      cashAccounts,
		Counterparties:    counterpartyAccounts,
		Assets:            assets,
		Transactions:      transactions,
	}
	report.PortfolioValue.Return = Percent((endBalance.linkedTWR/startBalance.linkedTWR - 1) * 100)

	return report, nil
}

func (r *ReviewReport) NetGains() Money {
	return r.PortfolioValue.End.Sub(r.PortfolioValue.Start).Sub(r.CashFlow)
}
func (r *ReviewReport) MarketChange() Money {
	return r.MarketValue.End.Sub(r.MarketValue.Start)
}
