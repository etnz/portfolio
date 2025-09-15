package portfolio

import (
	"fmt"
)

// CostBasisMethod defines the method for calculating cost basis.
type CostBasisMethod int

const (
	// AverageCost calculates the cost basis by averaging the cost of all shares.
	AverageCost CostBasisMethod = iota
	// FIFO (First-In, First-Out) calculates the cost basis by assuming the first shares purchased are the first ones sold.
	FIFO
)

func (m CostBasisMethod) String() string {
	switch m {
	case AverageCost:
		return "average"
	case FIFO:
		return "fifo"
	default:
		return "unknown"
	}
}

// ParseCostBasisMethod parses a string into a CostBasisMethod.
func ParseCostBasisMethod(s string) (CostBasisMethod, error) {
	switch s {
	case "average":
		return AverageCost, nil
	case "fifo":
		return FIFO, nil
	default:
		return 0, fmt.Errorf("unknown cost basis method: %q", s)
	}
}

// GainsReport contains the results of a capital gains calculation.
type GainsReport struct {
	Range             Range
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

// NewGainsReport computes the realized and unrealized gains for all securities
// over a given period, using a specified cost basis accounting method.
func NewGainsReport(ledger *Ledger, period Range, method CostBasisMethod) (*GainsReport, error) {
	journal := ledger.journal
	if journal == nil {
		return &GainsReport{}, nil
	}

	endBalance, err := NewBalance(journal, period.To, method)
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	startBalance, err := NewBalance(journal, period.From.Add(-1), method)
	if err != nil {
		return nil, fmt.Errorf("could not create balance from journal: %w", err)
	}
	return calculateGains(endBalance, startBalance, method, journal.cur)
}
func calculateGains(endBalance, startBalance *Balance, method CostBasisMethod, reportingCurrency string) (*GainsReport, error) {
	report := &GainsReport{
		Range:             Range{From: startBalance.on.Add(1), To: endBalance.on},
		Method:            method,
		ReportingCurrency: reportingCurrency,
		Securities:        []SecurityGains{},
	}
	// TotalPortfolio value is broken in three parts:
	// - the cash accounts (one per currency)
	// - the counterparty accounts (one per counterparty)
	// - the assets market value
	//
	// As a consequence there are three variations aka gains of those three parts.

	// gains are: total gain := change in total portfolio value
	//  market value gain:=  total gain - cash flow, and counterparty change
	// Completely independant: realized gain:
	// unrealized gains standing (does not depend on the period actually)
	totalRealized := M(0, reportingCurrency) // always in reporting currency, even 0
	for sec := range endBalance.Securities() {

		realizedGain := endBalance.RealizedGain(sec.Ticker()).Sub(startBalance.RealizedGain(sec.Ticker()))
		unrealizedGain := endBalance.MarketValue(sec.Ticker()).Sub(endBalance.CostBasis(sec.Ticker()))

		totalRealized = totalRealized.Add(endBalance.Convert(realizedGain))

		if realizedGain.IsZero() && unrealizedGain.IsZero() {
			continue
		}

		report.Securities = append(report.Securities, SecurityGains{
			Security:   sec.Ticker(),
			Realized:   realizedGain,
			Unrealized: unrealizedGain,
			Quantity:   endBalance.Position(sec.Ticker()),
		})
	}

	total := endBalance.TotalPortfolioValue().Sub(startBalance.TotalPortfolioValue())

	report.Total = total
	report.Realized = totalRealized
	report.Unrealized = endBalance.TotalUnrealizedGain()

	return report, nil
}
