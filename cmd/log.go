package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"slices"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

type logCmd struct {
	period string
	start  string
	date   string
	method string
}

func (*logCmd) Name() string { return "log" }
func (*logCmd) Synopsis() string {
	return "display a chronological log of all transactions and their impact on the portfolio"
}
func (*logCmd) Usage() string {
	return `pcs log [-p <period> | -s <start_date>] [-d <end_date>] [-method <method>]

  Generates a detailed, stateful log of all portfolio activities within a
  given date range, showing the impact of each transaction. The log is
  punctuated by periodic performance summaries.
`
}

func (p *logCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.period, "p", "month", "Predefined period for the log (day, week, month, quarter, year).")
	f.StringVar(&p.start, "s", "", "The start date for a custom log range. Overrides -p.")
	f.StringVar(&p.date, "d", "0d", "The end date for the log (defaults to today).")
	f.StringVar(&p.method, "method", "average", "The cost basis method (average, fifo) to use for calculating realized gains.")
}

func (p *logCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	var periodRange portfolio.Range
	endDate, err := portfolio.ParseDate(p.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing end date: %v\n", err)
		return subcommands.ExitFailure
	}

	if p.start != "" {
		startDate, err := portfolio.ParseDate(p.start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing start date: %v\n", err)
			return subcommands.ExitFailure
		}
		periodRange = portfolio.NewRange(startDate, endDate)
	} else {
		period, err := portfolio.ParsePeriod(p.period)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing period: %v\n", err)
			return subcommands.ExitFailure
		}
		periodRange = period.Range(endDate)
	}

	costMethod, err := portfolio.ParseCostBasisMethod(p.method)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	// TODO: the periodRange and period should work together.
	reviewBlocks, err := ledger.GenerateLog(periodRange, portfolio.Monthly)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	output, err := renderer.LogMarkdown(reviewBlocks, slices.Collect(ledger.AllSecurities()), costMethod)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	// review, err := ledger.NewReview(periodRange)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	return subcommands.ExitFailure
	// }
	// output, err := renderer.LogMarkdown(review, slices.Collect(ledger.HeldSecuritiesInRange(periodRange)), costMethod)
	// if err != nil {
	// 	fmt.Fprintln(os.Stderr, err)
	// 	return subcommands.ExitFailure
	// }

	printMarkdown(output)

	return subcommands.ExitSuccess
}
