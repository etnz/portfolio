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
	inception bool
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
}

func (c *amundiFetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	// load headers
	headers, err := amundi.LoadHeaders()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: not logged in to Amundi, run amundi login first: %v\n", err)
		return subcommands.ExitFailure
	}

	// Load the ledger.
	// TODO: the DecodeLedger will work only as long as this command is part of the pcs CLI
	//       once it is exported as an extension, a different function will have to be implemented.
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	changes, updates, err := amundi.Fetch(headers, ledger, c.inception)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not fetch from Amundi: %v\n", err)
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

	err = portfolio.EncodeLedger(file, ledger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error wrting  updated ledger file: %v\n", err)
		return subcommands.ExitFailure
	}

	for _, c := range changes {
		if c.Old != nil {
			fmt.Printf("%s %15s: %10s (%10s)\n", c.Date, c.ID, c.Old.StringFixed(4), c.New.StringFixed(4))
		} else {
			fmt.Printf("%s %15s: %10s\n", c.Date, c.ID, c.New.StringFixed(4))
		}
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully fetched from Amundi.\n")
	return subcommands.ExitSuccess
}
