package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type fmtCmd struct {
	outputFile string
	ledgerFile string
}

func (*fmtCmd) Name() string { return "fmt" }
func (*fmtCmd) Synopsis() string {
	return "validates and formats the ledger file into a canonical form"
}
func (*fmtCmd) Usage() string {
	return `pcs fmt [-l <ledger_name>]

  Validates and formats the ledger file. This command reads all transactions,
  validates them, applies available quick-fixes (like resolving "sell all"),
  sorts them by date, and writes them back in a canonical JSONL format.
  By default, it formats all ledgers in-place. Use -l to specify a single ledger.

Usage Examples:
# Writes to the default ledger file.
$ pcs fmt

`
}

func (p *fmtCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.ledgerFile, "l", "", "Ledger to format. Formats all by default.")
}

func (p *fmtCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ledgers, err := DecodeLedgers(p.ledgerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ledgers: %v\n", err)
		return subcommands.ExitFailure
	}

	if len(ledgers) == 0 {
		fmt.Fprintf(os.Stderr, "Warning: no ledgers found to format.\n")
		return subcommands.ExitSuccess
	}

	// Default case: format all specified ledgers in-place
	for _, ledger := range ledgers {
		ledgerName := ledger.Name()
		fmt.Fprintf(os.Stderr, "Formatting ledger %q...\n", ledgerName)

		formattedLedger, err := ledger.Fmt()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting ledger %q: %v\n", ledgerName, err)
			continue
		}

		if err := portfolio.SaveLedger(PortfolioPath(), formattedLedger); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving formatted ledger %q: %v\n", ledgerName, err)
			continue
		}
		fmt.Fprintf(os.Stderr, "Finished formatting ledger %q.\n", ledgerName)
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully formatted ledgers.\n")
	return subcommands.ExitSuccess
}
