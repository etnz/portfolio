package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

type summaryCmd struct {
	date     string
	currency string
}

func (*summaryCmd) Name() string     { return "summary" }
func (*summaryCmd) Synopsis() string { return "display portfolio summary dashboard" }
func (*summaryCmd) Usage() string {
	return `summary [-d <date>] [-c <currency>]

  Displays a summary of the portfolio, including total market value.
`
}

func (c *summaryCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the summary (YYYY-MM-DD)")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for the summary")
}

func (c *summaryCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	market, err := DecodeSecurities()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities: %v\n", err)
		return subcommands.ExitFailure
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	as, err := portfolio.NewAccountingSystem(ledger, market, c.currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	summary, err := as.NewSummary(on)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating portfolio summary: %v\n", err)
		return subcommands.ExitFailure
	}

	// Helper to format performance percentages
	formatPerf := func(p portfolio.Performance) string {
		if p.StartValue == 0 {
			return "N/A"
		}
		return fmt.Sprintf("%+.2f%%", p.Return*100)
	}

	_, week := summary.Date.ISOWeek()
	quarter := (summary.Date.Month()-1)/3 + 1

	dayLabel := fmt.Sprintf("Day %d:", summary.Date.Day())
	weekLabel := fmt.Sprintf("Week %d:", week)
	monthLabel := fmt.Sprintf("%s:", summary.Date.Month())
	quarterLabel := fmt.Sprintf("Q%d:", quarter)
	yearLabel := fmt.Sprintf("%d:", summary.Date.Year())

	fmt.Printf("Portfolio Summary on %s\n", summary.Date)
	fmt.Println("-------------------------------------------")
	fmt.Printf("Total Market Value: %.2f %s\n", summary.TotalMarketValue, summary.ReportingCurrency)
	fmt.Println()
	fmt.Println("Performance:")
	fmt.Printf("  %-11s %10s\n", dayLabel, formatPerf(summary.Daily))
	fmt.Printf("  %-11s %10s\n", weekLabel, formatPerf(summary.WTD))
	fmt.Printf("  %-11s %10s\n", monthLabel, formatPerf(summary.MTD))
	fmt.Printf("  %-11s %10s\n", quarterLabel, formatPerf(summary.QTD))
	fmt.Printf("  %-11s %10s\n", yearLabel, formatPerf(summary.YTD))
	fmt.Printf("  %-11s %10s\n", "Inception:", formatPerf(summary.Inception))

	return subcommands.ExitSuccess
}
