package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/insee"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

// inseeFetchCmd implements the "insee fetch" command.
type inseeFetchCmd struct {
	inception bool
}

func (*inseeFetchCmd) Name() string     { return "fetch" }
func (*inseeFetchCmd) Synopsis() string { return "fetches market data from INSEE" }
func (*inseeFetchCmd) Usage() string {
	return `insee fetch:

Fetches market data from data.insee.fr.
`
}

func (c *inseeFetchCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&c.inception, "inception", false, "ignore existing prices in ledger, and fetch all from inception date")
}

func (c *inseeFetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	updates, err := insee.Fetch(ledger, c.inception)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not fetch from data.insee.fr: %v\n", err)
		return subcommands.ExitFailure
	}

	// Update the ledger with the new market data.
	if _, err := ledger.UpdateMarketData(updates...); err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not add market data to the ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	file, err := os.Create(*ledgerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening ledger file %q for writing: %v\n", *ledgerFile, err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	if err := portfolio.EncodeLedger(file, ledger); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing updated ledger file: %v\n", err)
		return subcommands.ExitFailure
	}

	for _, upd := range updates {
		fmt.Println(upd.When(), renderer.Transaction(upd))
	}
	if len(updates) == 0 {
		fmt.Fprintf(os.Stderr, "✅ Successfully fetched from data.insee.fr but there are no new data.\n")
	} else {
		fmt.Fprintf(os.Stderr, "✅ Successfully fetched from data.insee.fr and updated ledger.\n")
	}
	return subcommands.ExitSuccess
}
