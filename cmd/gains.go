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
	period     string
	start      string
	end        string
	method     string
	update     bool
	ledgerFile string
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
	f.StringVar(&c.method, "method", "average", "Cost basis method (average, fifo)")
	f.StringVar(&c.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
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

	ledger, err := DecodeLedger(c.ledgerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledger %q: %v\n", c.ledgerFile, err)
		return subcommands.ExitFailure
	}

	if c.update {
		ledger.UpdateIntraday()
	}

	// Parse cost basis method
	method, err := portfolio.ParseCostBasisMethod(c.method)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing cost basis method: %v\n", err)
		return subcommands.ExitUsageError
	}

	// Calculate gains
	review, err := ledger.NewReview(period)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating gains: %v\n", err)
		return subcommands.ExitFailure
	}

	// Print report
	md := renderer.GainsMarkdown(review, method)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
