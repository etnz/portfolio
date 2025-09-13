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

// reviewCmd holds the flags for the 'review' subcommand.
type reviewCmd struct {
	period string
	date   string
	start  string
	update bool
}

func (*reviewCmd) Name() string     { return "review" }
func (*reviewCmd) Synopsis() string { return "review a portfolio performance" }
func (*reviewCmd) Usage() string {
	return `pcs review [-period <period>| -s <date>] [-d <date>]
	
  Review the portfolio transactions for a given period.
`
}

func (c *reviewCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Date for the report. See the user manual for supported date formats.")
	f.StringVar(&c.period, "period", portfolio.Daily.String(), "period for the review (day, week, month, quarter, year)")
	f.StringVar(&c.start, "s", "", "Start date of the reporting period. Overrides -period.")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before generating the report")
}

func (c *reviewCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// 1. Parse the date range for the report
	var r portfolio.Range
	endDate, err := portfolio.ParseDate(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if c.start != "" {
		// Custom range using start and end dates
		startDate, err := portfolio.ParseDate(c.start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing start date: %v\n", err)
			return subcommands.ExitUsageError
		}
		r = portfolio.Range{From: startDate, To: endDate}
	} else {
		// Predefined period
		p, err := portfolio.ParsePeriod(c.period)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Invalid period: %v\n", err)
			return subcommands.ExitUsageError
		}
		r = p.Range(endDate)
	}

	// Truncate the range if it goes beyond the present day.
	today := portfolio.Today()
	if r.To.After(today) {
		r.To = today
	}

	if r.To.IsToday() {
		c.update = true
	}

	// 2. Decode the ledger
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		// if err := as.MarketData.UpdateIntraday(); err != nil {
		// 	fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
		// 	// We can continue with stale prices, so this is not a fatal error.
		// }
	}

	// 3. Generate the report
	report, err := portfolio.NewReviewReport(ledger, *defaultCurrency, r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating review report: %v\n", err)
		return subcommands.ExitFailure
	}

	// 4. Render the report
	md := renderer.ReviewMarkdown(report)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
