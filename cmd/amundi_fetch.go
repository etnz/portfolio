package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/amundi"
	"github.com/google/subcommands"
)

// amundiFetchCmd implements the "amundi fetch" command.
type amundiFetchCmd struct {
	inception  bool
	ledgerFile string
}

func (*amundiFetchCmd) Name() string     { return "fetch" }
func (*amundiFetchCmd) Synopsis() string { return "fetches market data from Amundi" }
func (*amundiFetchCmd) Usage() string {
	return `amundi fetch

Fetches holding positions details for all Amundi products and update ledger prices.
It only update prices for asset actually declared in the ledger.
It uses prices found in today's holding position report, and as long as there a new prices,
it continues backward.
`
}

func (c *amundiFetchCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&c.inception, "inception", false, "ignore existing prices in ledger, and fetch all from today back to inception date")
	f.StringVar(&c.ledgerFile, "l", "", "ledger name to update. update all ledgers by default.")
}

func (c *amundiFetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	// load headers
	headers, err := amundi.LoadHeaders()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: not logged in to Amundi, run amundi login first: %v\n", err)
		return subcommands.ExitFailure
	}

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
		changes, updates, err := amundi.Fetch(headers, ledger, c.inception)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not fetch from Amundi for ledger %q: %v\n", ledgerName, err)
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

		for _, c := range changes {
			if c.Old != nil {
				fmt.Printf("%s %15s: %10s (%10s)\n", c.Date, c.ID, c.Old.StringFixed(4), c.New.StringFixed(4))
			} else {
				fmt.Printf("%s %15s: %10s\n", c.Date, c.ID, c.New.StringFixed(4))
			}
		}
		if summary.Total() > 0 {
			fmt.Fprintf(os.Stderr, "Finished processing ledger %q, %d updates applied.\n", ledgerName, summary.Total())
		} else {
			fmt.Fprintf(os.Stderr, "Finished processing ledger %q, no updates.\n", ledgerName)
		}
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully fetched from Amundi and updated ledgers.\n")
	return subcommands.ExitSuccess
}
