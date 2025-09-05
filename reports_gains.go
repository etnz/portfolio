package portfolio

import (
	"fmt"

	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

// GainsReport contains the results of a capital gains calculation.
type GainsReport struct {
	Range             date.Range
	Method            CostBasisMethod
	ReportingCurrency string
	Securities        []SecurityGains
	Realized          Money
	Unrealized        Money
	Total             Money
}

// SecurityGains holds the realized and unrealized gains for a single security.
type SecurityGains struct {
	Security   string
	Realized   Money
	Unrealized Money
	Quantity   Quantity
}

// CalculateGains computes the realized and unrealized gains for all securities
// over a given period, using a specified cost basis accounting method.
func (as *AccountingSystem) CalculateGains(period date.Range, method CostBasisMethod) (*GainsReport, error) {
	report := &GainsReport{
		Range:             period,
		Method:            method,
		ReportingCurrency: as.ReportingCurrency,
		Securities:        []SecurityGains{},
	}

	journal, err := as.getJournal()
	if err != nil {
		return nil, fmt.Errorf("could not get journal: %w", err)
	}

	endBalance, err := NewBalance(journal, period.To, method)
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	startBalance, err := NewBalance(journal, period.From.Add(-1), method)
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	// gains are: total gain := change in total portfolio value
	//  market value gain:=  total gain - cash flow, and counterparty change
	// Completely independant: realized gain:
	// unrealized gains standing (does not depend on the period actually)

	totalRealized := decimal.Zero
	for sec := range endBalance.Securities() {

		realizedGain := endBalance.RealizedGain(sec.Ticker()).Sub(startBalance.RealizedGain(sec.Ticker()))
		unrealizedGain := endBalance.MarketValue(sec.Ticker()).Sub(endBalance.CostBasis(sec.Ticker()))

		totalRealized = totalRealized.Add(endBalance.Convert(realizedGain, sec.currency))

		if realizedGain.IsZero() && unrealizedGain.IsZero() {
			continue
		}

		report.Securities = append(report.Securities, SecurityGains{
			Security:   sec.Ticker(),
			Realized:   NewMoney(realizedGain, sec.currency),
			Unrealized: NewMoney(unrealizedGain, sec.currency),
			Quantity:   NewQuantity(endBalance.Position(sec.Ticker())),
		})
	}

	total := endBalance.TotalPortfolioValue().Sub(startBalance.TotalPortfolioValue())

	report.Total = NewMoney(total, as.ReportingCurrency)
	report.Realized = NewMoney(totalRealized, as.ReportingCurrency)
	report.Unrealized = NewMoney(endBalance.TotalUnrealizedGain(), as.ReportingCurrency)

	return report, nil
}
