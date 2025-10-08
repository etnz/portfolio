package renderer

import (
	"fmt"
	"io"

	"github.com/etnz/portfolio"
)

func getPeriodRanges(p portfolio.Period, on portfolio.Date) (current portfolio.Range, previous portfolio.Range) {
	// The current range is from the start of the period to the given date 'on'.
	current = on.Range(p)

	// The previous range is the full period range for the day just before the current one starts.
	previous = on.StartOf(p).Add(-1).Range(p)

	return
}

func RenderMultiPeriodSummary(w io.Writer, on portfolio.Date, ledger *portfolio.Ledger) bool {
	// Periods to be displayed
	periods := []portfolio.Period{
		portfolio.Daily,
		portfolio.Weekly,
		portfolio.Monthly,
		portfolio.Quarterly,
		portfolio.Yearly,
	}

	// but each period lead to different ranges, one for the previous period one for the current period.
	var reviews []*portfolio.Review
	for _, p := range periods {
		currentRange, previousRange := getPeriodRanges(p, on)

		// Add previous period
		prevReview := ledger.NewReview(previousRange)
		reviews = append(reviews, prevReview)

		// Add current period
		currentReview := ledger.NewReview(currentRange)
		reviews = append(reviews, currentReview)
	}

	fmt.Fprintf(w, "# Portfolio Summary on %s\n\n", on)
	fmt.Fprintf(w, "*As of %s*\n\n", Now().Format("2006-01-02 15:04:05"))

	// Header
	fmt.Fprint(w, "| |")
	for _, r := range reviews {
		fmt.Fprintf(w, " %s |", r.Range().Identifier())
	}
	fmt.Fprintln(w, "")

	// Separator
	fmt.Fprint(w, "|:---|")
	for range reviews {
		fmt.Fprint(w, "---:|")
	}
	fmt.Fprintln(w, "")

	// Rows

	printRow := func(label string, getValue func(r *portfolio.Review) string) {
		fmt.Fprintf(w, "| %s ", label)
		for _, r := range reviews {
			fmt.Fprintf(w, " | %s", getValue(r))
		}
		fmt.Fprintln(w, " |")
	}

	printLine := func() {
		fmt.Fprint(w, "| ")
		for range reviews {
			fmt.Fprintf(w, " | ")
		}
		fmt.Fprintln(w, " |")
	}

	printRowBold := func(label string, getValue func(r *portfolio.Review) string) {
		fmt.Fprintf(w, "| **%s** ", label)
		for _, r := range reviews {
			fmt.Fprintf(w, " | **%s** ", getValue(r))
		}
		fmt.Fprintln(w, " |")
	}

	printRowBold("Total Portfolio Value", func(r *portfolio.Review) string { return r.End().TotalPortfolio().String() })
	printRow("Previous Value", func(r *portfolio.Review) string { return r.Start().TotalPortfolio().SignedString() })
	printLine()
	printRow("\u00A0\u00A0Capital Flow", func(r *portfolio.Review) string { return r.CashFlow().SignedString() })
	printRow("+ Market Gains", func(r *portfolio.Review) string { return r.MarketGain().SignedString() })
	printRow("+ Forex Gains", func(r *portfolio.Review) string {
		return r.PortfolioChange().Sub(r.CashFlow()).Sub(r.MarketGain()).SignedString()
	})
	printRowBold("= Net Change", func(r *portfolio.Review) string { return r.PortfolioChange().SignedString() })
	printLine()

	// This section is printed only if there's more than one component of change.
	// We check across all reviews to see if it's worth printing.
	shouldPrintChangeBreakdown := false
	for _, r := range reviews {
		if !r.CashChange().IsZero() || !r.CounterpartyChange().IsZero() || !r.TotalMarketChange().IsZero() {
			shouldPrintChangeBreakdown = true
			break
		}
	}
	if shouldPrintChangeBreakdown {
		printLine()
		printRow("\u00A0\u00A0Cash Change", func(r *portfolio.Review) string { return r.CashChange().SignedString() })
		printRow("+ Counterparties Change", func(r *portfolio.Review) string { return r.CounterpartyChange().SignedString() })
		printRow("+ Market Value Change", func(r *portfolio.Review) string { return r.TotalMarketChange().SignedString() })
		printRowBold("= Net Change", func(r *portfolio.Review) string { return r.PortfolioChange().SignedString() })
	}

	printLine()
	printRow("\u00A0\u00A0Dividends", func(r *portfolio.Review) string { return r.Dividends().SignedString() })
	printRow("+ Market Gains", func(r *portfolio.Review) string { return r.MarketGain().SignedString() })
	printRow("+ Forex Gains", func(r *portfolio.Review) string {
		return r.PortfolioChange().Sub(r.CashFlow()).Sub(r.MarketGain()).SignedString()
	})
	printRowBold("= Total Gains", func(r *portfolio.Review) string {
		forexGain := r.PortfolioChange().Sub(r.CashFlow()).Sub(r.MarketGain())
		return r.MarketGain().Add(forexGain).Add(r.Dividends()).SignedString()
	})

	return true
}
