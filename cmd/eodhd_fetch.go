package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/eodhd"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

const eodhd_api_key = "EODHD_API_KEY"

// eodhdFetchCmd implements the "eodhd fetch" command.
type eodhdFetchCmd struct {
	eodhdApiFlag string
	inception    bool
}

func (*eodhdFetchCmd) Name() string     { return "fetch" }
func (*eodhdFetchCmd) Synopsis() string { return "fetches market data from EODHD" }
func (*eodhdFetchCmd) Usage() string {
	return `eodhd fetch:
	
	Fetches market data from eodhd.com.

	Requires the EODHD_API_TOKEN environment variable to be set or passed as a flag.
	`
}
func (c *eodhdFetchCmd) SetFlags(f *flag.FlagSet) {
	flag.StringVar(&c.eodhdApiFlag, "eodhd-api-key", "", "EODHD API key to use for consuming EODHD.com API. This flag takes precedence over the "+eodhd_api_key+" environment variable. You can get one at https://eodhd.com/")
	flag.BoolVar(&c.inception, "inception", false, "ignore existing prices in ledger, and fetch all from inception date")
}

// eodhdApiKey retrieves the EODHD API key from the command-line flag or the environment variable.
// It prioritizes the flag over the environment variable.
func (c *eodhdFetchCmd) eodhdApiKey() string {
	// If the flag is not set, we try to read it from the environment variable.
	if c.eodhdApiFlag == "" {
		c.eodhdApiFlag = os.Getenv(eodhd_api_key)
	}
	return c.eodhdApiFlag
}
func (c *eodhdFetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// Load the ledger.
	// TODO: the DecodeLedger will work only as long as this command is part of the pcs CLI
	//       once it is exported as an extension, a different function will have to be implemented.

	key := c.eodhdApiKey()
	if key == "" {
		fmt.Fprintf(os.Stderr, "Error: EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable\n")
		return subcommands.ExitFailure
	}

	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	_, updates, err := eodhd.Fetch(key, ledger, c.inception)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not fetch from eodhd.com: %v\n", err)
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

	for _, upd := range updates {
		fmt.Println(upd.When(), renderer.Transaction(upd))
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully fetched from eodhd.com and updated ledger.\n")
	return subcommands.ExitSuccess
}
