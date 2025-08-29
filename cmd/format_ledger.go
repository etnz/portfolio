package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type formatLedgerCmd struct{}

func (*formatLedgerCmd) Name() string     { return "format-ledger" }
func (*formatLedgerCmd) Synopsis() string { return "formats the ledger file into a canonical form" }
func (*formatLedgerCmd) Usage() string {
	return `format-ledger:
  formats the ledger file into a canonical form.
`
}

func (p *formatLedgerCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(ledgerFile, "ledger-file", "transactions.jsonl", "Path to the ledger file containing transactions (JSONL format)")
}

func (p *formatLedgerCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// 1. Read the ledger
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	// 2. Write the ledger back to the same file
	err = EncodeLedger(ledger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Ledger file '%s' has been formatted.\n", *ledgerFile)
	return subcommands.ExitSuccess
}

// EncodeLedger encodes the ledger to the application's default ledger file.
// This function is new and will be implemented here in cmd package.
func EncodeLedger(ledger *portfolio.Ledger) error {
	f, err := os.OpenFile(*ledgerFile, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening ledger file %q for writing: %w", *ledgerFile, err)
	}
	defer f.Close()

	return portfolio.EncodeLedger(f, ledger)
}
