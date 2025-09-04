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

// summaryCmd holds the flags for the 'summary' subcommand.
type summaryCmd struct {
	date     string
	currency string
	update   bool
}

func (*summaryCmd) Name() string     { return "summary" }
func (*summaryCmd) Synopsis() string { return "display a portfolio performance summary" }
func (*summaryCmd) Usage() string {
	return `pcs summary [-d <date>] [-c <currency>] [-u]

  Displays a summary of the portfolio, including total market value.
`
}

func (c *summaryCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the summary. See the user manual for supported date formats.")
	f.StringVar(&c.currency, "c", "EUR", "Reporting currency for the summary")
	f.BoolVar(&c.update, "u", false, "update with latest intraday prices before calculating summary")
}

func (c *summaryCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	on, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}

	if on.IsToday() {
		c.update = true
	}

	as, err := DecodeAccountingSystem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		err := as.MarketData.UpdateIntraday()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	summary, err := as.NewSummary(on)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating portfolio summary: %v\n", err)
		return subcommands.ExitFailure
	}

	md := renderer.SummaryMarkdown(summary)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
