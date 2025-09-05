package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio/date"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// reviewCmd holds the flags for the 'review' subcommand.
type reviewCmd struct {
	period string
	date   string
}

func (*reviewCmd) Name() string     { return "review" }
func (*reviewCmd) Synopsis() string { return "review a portfolio performance" }
func (*reviewCmd) Usage() string {
	return `pcs review [-period <period>] [-d <date>]

  Review the portfolio transactions for a given period.
`
}

func (c *reviewCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the report. See the user manual for supported date formats.")
	f.StringVar(&c.period, "period", "month", "period for the review (week, month, quarter, year)")
}

func (c *reviewCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// 1. Parse the date range for the report
	var r date.Range
	var err error
	on, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	switch c.period {
	case "week":
		r = date.NewRangeFrom(on, c.period)

	case "month":
		r = date.NewRangeFrom(on, c.period)

	case "quarter":
		r = date.NewRangeFrom(on, c.period)

	case "year":
		r = date.NewRangeFrom(on, c.period)
	default:
		fmt.Fprintf(os.Stderr, "Invalid period: %s\n", c.period)
		return subcommands.ExitUsageError
	}

	// 2. Create the accounting system
	as, err := DecodeAccountingSystem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	// 3. Generate the report
	report, err := as.NewReviewReport(r)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating review report: %v\n", err)
		return subcommands.ExitFailure
	}

	// 4. Render the report
	md := renderer.ReviewMarkdown(report)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
