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

// summaryCmd holds the flags for the 'summary' subcommand.
type summaryCmd struct {
	date   string
	update bool
}

func (*summaryCmd) Name() string     { return "summary" }
func (*summaryCmd) Synopsis() string { return "display a portfolio performance summary" }
func (*summaryCmd) Usage() string {
	return `pcs summary [-d <date>]

  Displays a summary of the portfolio, including total market value.
`
}

func (c *summaryCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Date for the summary. See the user manual for supported date formats.")
}

func (c *summaryCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := portfolio.ParseDate(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if on.IsToday() {
		c.update = true
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		// err := as.MarketData.UpdateIntraday()
		// if err != nil {
		// 	fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
		// 	return subcommands.ExitFailure
		// }
	}

	summary, err := portfolio.NewSummary(ledger, on)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating portfolio summary: %v\n", err)
		return subcommands.ExitFailure
	}

	md := renderer.SummaryMarkdown(summary)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
