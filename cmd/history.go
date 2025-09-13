package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// historyCmd holds the flags for the 'history' subcommand.
type historyCmd struct {
	security string
	currency string
}

func (*historyCmd) Name() string     { return "history" }
func (*historyCmd) Synopsis() string { return "display asset value history" }
func (*historyCmd) Usage() string {
	return `pcs history -s <security> | -c <currency>\n\n  Displays the value of a single asset or cash account over time.\n`
}

func (c *historyCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.security, "s", "", "security ticker to report on")
	f.StringVar(&c.currency, "c", "", "currency of cash account to report on")
}

func (c *historyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if (c.security == "" && c.currency == "") || (c.security != "" && c.currency != "") {
		fmt.Fprintln(os.Stderr, "either -s or -c must be provided")
		return subcommands.ExitUsageError
	}

	ledger, err := DecodeLedger()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintln(os.Stderr, "Ledger file not found. Nothing to report.")
			return subcommands.ExitSuccess
		}
		fmt.Fprintf(os.Stderr, "Error decoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	// The reporting currency for history is based on the asset's own currency.
	report, err := portfolio.NewHistory(ledger, c.security, c.currency, *defaultCurrency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating history report: %v\n", err)
		return subcommands.ExitFailure
	}

	md := renderer.HistoryMarkdown(report)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
