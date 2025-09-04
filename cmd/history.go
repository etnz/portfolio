package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

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

	as, err := DecodeAccountingSystem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	report, err := as.NewHistory(c.security, c.currency)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error calculating history: %v\n", err)
		return subcommands.ExitFailure
	}

	md := renderer.HistoryMarkdown(report)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
