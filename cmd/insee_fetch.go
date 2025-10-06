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
	inception  bool
	ledgerFile string
}

func (*inseeFetchCmd) Name() string     { return "fetch" }
func (*inseeFetchCmd) Synopsis() string { return "fetches market data from INSEE" }
func (*inseeFetchCmd) Usage() string {
	return `insee fetch:

Fetches market data from data.insee.fr.
`
}

func (c *inseeFetchCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ledgerFile, "l", "", "Ledger name to update. Updates all ledgers by default.")
	f.BoolVar(&c.inception, "inception", false, "ignore existing prices in ledger, and fetch all from inception date")
}

func (c *inseeFetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ledgers, err := DecodeLedgers(c.ledgerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ledgers: %v\n", err)
		return subcommands.ExitFailure
	}

	if len(ledgers) == 0 {
		fmt.Fprintf(os.Stderr, "Warning: no ledgers found to update.\n")
		return subcommands.ExitSuccess
	}

	for _, ledger := range ledgers {
		ledgerName := ledger.Name()
		fmt.Fprintf(os.Stderr, "Processing ledger %q...\n", ledgerName)
		updates, err := insee.Fetch(ledger, c.inception)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not fetch from data.insee.fr for ledger %q: %v\n", ledgerName, err)
			continue
		}

		// Update the ledger with the new market data.
		summary, err := ledger.UpdateMarketData(updates...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not add market data to the ledger %q: %v\n", ledgerName, err)
			continue
		}

		if err := portfolio.SaveLedger(PortfolioPath(), ledger); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing updated ledger file for %q: %v\n", ledgerName, err)
			continue
		}

		for _, upd := range updates {
			fmt.Println(upd.When(), renderer.Transaction(upd))
		}
		if summary.Total() > 0 {
			fmt.Fprintf(os.Stderr, "Finished processing ledger %q, %d updates applied.\n", ledgerName, summary.Total())
		} else {
			fmt.Fprintf(os.Stderr, "Finished processing ledger %q, no updates.\n", ledgerName)
		}
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully fetched from data.insee.fr and updated ledgers.\n")
	return subcommands.ExitSuccess
}
