package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// summaryCmd holds the flags for the 'summary' subcommand.
type summaryCmd struct {
	date       string
	ledgerFile string
	update     bool
}

func (*summaryCmd) Name() string     { return "summary" }
func (*summaryCmd) Synopsis() string { return "display a portfolio performance summary" }
func (*summaryCmd) Usage() string {
	return `pcs summary [-d <date>] [-l <ledger>]

  Displays a summary of the portfolio, including total market value.
`
}

func (c *summaryCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Date for the summary. See the user manual for supported date formats.")
	f.StringVar(&c.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
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

	ledger, err := DecodeLedger(c.ledgerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding ledger %q: %v\n", c.ledgerFile, err)
		return subcommands.ExitFailure
	}

	if c.update {
		err := ledger.UpdateIntraday()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error updating intraday prices: %v\n", err)
			return subcommands.ExitFailure
		}
	}

	var b strings.Builder

	renderer.RenderMultiPeriodSummary(&b, on, ledger.Journal())

	printMarkdown(b.String())

	return subcommands.ExitSuccess
}
