package portfolio

import (
	"fmt"

	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

// ReviewReport contains the transactions for a given period.
type ReviewReport struct {
	Range               date.Range
	ReportingCurrency   string
	TotalPortfolioValue Money
	PrevPortfolioValue  Money
	TotalMarketValue    Money
	PrevMarketValue     Money
	CashChange          Money // Variation of cash value.
	CashFlow            Money // Variation of cash value. TODO add support for cash flow
	CounterpartyChange  Money // Variation of Counterpary Value.
	Gains               *GainsReport
	Assets              []AssetReview
	Transactions        []Transaction
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

// NewReviewReport returns a report with all transactions in a given period.
func (as *AccountingSystem) NewReviewReport(p date.Range) (*ReviewReport, error) {
	period := p
	period.From = period.From.Add(-1)

	report := &ReviewReport{
		Range:             p,
		ReportingCurrency: as.ReportingCurrency,
		Transactions:      []Transaction{},
		Assets:            []AssetReview{},
	}

	// Transactions
	for _, tx := range as.Ledger.transactions {
		if p.Contains(tx.When()) {
			report.Transactions = append(report.Transactions, tx)
		}
	}

	// Summary
	endBalance, err := as.Balance(period.To)
	if err != nil {
		return nil, err
	}

	startBalance, err := as.Balance(period.From.Add(-1))
	if err != nil {
		return nil, err
	}
	// Calculate current total portfolio value.
	report.TotalPortfolioValue = NewMoney(endBalance.TotalPortfolioValue(), as.ReportingCurrency)
	report.PrevPortfolioValue = NewMoney(startBalance.TotalPortfolioValue(), as.ReportingCurrency)
	report.TotalMarketValue = NewMoney(endBalance.TotalMarketValue(), as.ReportingCurrency)
	report.PrevMarketValue = NewMoney(startBalance.TotalMarketValue(), as.ReportingCurrency)
	report.CashChange = NewMoney(endBalance.TotalCash().Sub(startBalance.TotalCash()), as.ReportingCurrency)

	totalCashFlow := decimal.Zero
	for cur := range endBalance.Currencies() {
		totalCashFlow = totalCashFlow.Add(endBalance.CashFlow(cur).Sub(startBalance.CashFlow(cur)))
	}
	report.CashFlow = NewMoney(totalCashFlow, as.ReportingCurrency)

	totalCounterparties := decimal.Zero
	for acc := range endBalance.Counterparties() {
		cur := endBalance.CounterpartyCurrency(acc)
		totalCounterparties = totalCounterparties.Add(endBalance.Counterparty(cur).Sub(startBalance.Counterparty(cur)))
	}
	report.CounterpartyChange = NewMoney(totalCounterparties, as.ReportingCurrency)

	// Gains
	report.Gains, err = as.CalculateGains(p, FIFO)
	if err != nil {
		return nil, fmt.Errorf("could not calculate gains for review report: %w", err)
	}

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

		if startPos.IsZero() && endPos.IsZero() && realizedGain.IsZero() {
			continue
		}
		report.Assets = append(report.Assets, AssetReview{
			Security:         ticker,
			StartingPosition: NewQuantity(startPos),
			StartingValue:    NewMoney(startValue, currency),
			EndingPosition:   NewQuantity(endPos),
			EndingValue:      NewMoney(endValue, currency),
			RealizedGains:    NewMoney(realizedGain, currency),
			UnrealizedGains:  NewMoney(unrealizedGain, currency),
		})
	}

	return report, nil
}

func (r *ReviewReport) MarketChange() Money {
	return r.TotalMarketValue.Sub(r.PrevMarketValue).Sub(r.CashFlow)
}
