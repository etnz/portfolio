package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"slices"

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

	var predicate func(portfolio.Transaction) bool
	if c.security != "" {
		predicate = portfolio.BySecurity(c.security)
	} else {
		predicate = ledger.ByCurrency(c.currency)
	}

	// Build a list of all unique days where there was a significant transaction.
	dates := make(map[portfolio.Date]struct{})
	for _, tx := range ledger.Transactions(predicate) {
		dates[tx.When()] = struct{}{}
	}

	// Convert map keys to a slice and sort them.
	sortedDates := make([]portfolio.Date, 0, len(dates))
	for d := range dates {
		sortedDates = append(sortedDates, d)
	}
	slices.SortFunc(sortedDates, func(a, b portfolio.Date) int {
		if a.Before(b) {
			return -1
		}
		return 1
	})

	snapshots := make([]*portfolio.Snapshot, 0, len(sortedDates))
	for _, on := range sortedDates {
		s := ledger.NewSnapshot(on)
		snapshots = append(snapshots, s)
	}

	md := renderer.HistoryMarkdown(snapshots, c.security, c.currency)
	printMarkdown(md)

	return subcommands.ExitSuccess
}
