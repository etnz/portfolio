package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// gainsCmd holds the flags for the 'gains' subcommand.
type gainsCmd struct {
	period   string
	start    string
	end      string
	currency string
	method   string
	update   bool
}

func (*gainsCmd) Name() string     { return "gains" }
func (*gainsCmd) Synopsis() string { return "realized and unrealized gain analysis" }
func (*gainsCmd) Usage() string {
	return `pcs gains [-period <period>] [-s <date>] [-d <date>] [-c <currency>] [-method <method>] [-u]

  Calculates and displays realized and unrealized gains for each security.
`
}

func (c *gainsCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.period, "period", portfolio.Monthly.String(), "Predefined period (day, week, month, quarter, year)")
	f.StringVar(&c.start, "s", "", "Start date of the reporting period. See the user manual for supported date formats.")
	f.StringVar(&c.end, "d", portfolio.Today().String(), "End date of the reporting period. See the user manual for supported date formats.")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency")
	f.StringVar(&c.method, "method", "average", "Cost basis method (average, fifo)")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before calculating gains")
}

func (c *gainsCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.start != "" && c.period != "" {
		fmt.Fprintln(os.Stderr, "-start and -period flags cannot be used together")
		return subcommands.ExitUsageError
	}

	// Determine the reporting period
	var period portfolio.Range
	endDate, err := portfolio.ParseDate(c.end)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing end date: %v\n", err)
		return subcommands.ExitUsageError
	}
	p, err := portfolio.ParsePeriod(c.period)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing period: %v\n", err)
		return subcommands.ExitUsageError
	}

	if endDate.IsToday() {
		c.update = true
	}

	if c.start != "" {
		// Special range
		startDate, err := portfolio.ParseDate(c.start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing start date: %v\n", err)
			return subcommands.ExitUsageError
		}
		period = portfolio.Range{From: startDate, To: endDate}
	} else {
		// standard range
		period = p.Range(endDate)
	}

	// TODO: re-implement update logic with `fetch`
	if c.update {
		// err := market.UpdateIntraday()
		// if err != nil {
		// 	fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
		// 	return subcommands.ExitFailure
		// }
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	// Parse cost basis method
	method, err := portfolio.ParseCostBasisMethod(c.method)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing cost basis method: %v\n", err)
		return subcommands.ExitUsageError
	}

	// Calculate gains
	report, err := portfolio.NewGainsReport(ledger, c.currency, period, method)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating gains: %v\n", err)
		return subcommands.ExitFailure
	}

	// Print report
	md := renderer.GainsMarkdown(report)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
