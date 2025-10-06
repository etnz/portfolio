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

// holdingCmd holds the flags for the 'holding' subcommand.
type holdingCmd struct {
	date       string
	update     bool
	ledgerFile string
}

func (*holdingCmd) Name() string     { return "holding" }
func (*holdingCmd) Synopsis() string { return "displays portfolio holdings on a specific date" }
func (*holdingCmd) Usage() string {
	return `pcs holding [-d <date>] [-l <ledger>]

  Displays the portfolio's holdings (positions and cash balances) as of a specific date.
`
}

func (c *holdingCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.date, "d", portfolio.Today().String(), "Date for the holdings report. See the user manual for supported date formats.")
	f.StringVar(&c.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
}

func (c *holdingCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
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
		fmt.Fprintf(os.Stderr, "Error loading ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	if c.update {
		err := ledger.UpdateIntraday()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not update intraday prices: %v\n", err)
			// Continue without failing
		}
	}

	snapshot := ledger.NewSnapshot(on)

	printMarkdown(renderer.HoldingMarkdown(snapshot))

	return subcommands.ExitSuccess
}
