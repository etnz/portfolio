package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/eodhd"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

const eodhd_api_key = "EODHD_API_KEY"

type stringSliceFlag []string

// eodhdFetchCmd implements the "eodhd fetch" command.
type eodhdFetchCmd struct {
	eodhdApiFlag   string
	ledgerFile     string
	inception      bool
	tickers        stringSliceFlag
	fetchForex     bool
	fetchMSSI      bool
	fetchISIN      bool
	fetchPrices    bool
	fetchSplits    bool
	fetchDividends bool
}

func (f *stringSliceFlag) String() string { return strings.Join(*f, ", ") }
func (f *stringSliceFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
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
	f.StringVar(&c.ledgerFile, "l", "", "ledger name to update. update all ledgers by default.")
	f.StringVar(&c.eodhdApiFlag, "eodhd-api-key", "", "EODHD API key to use for consuming EODHD.com API. This flag takes precedence over the "+eodhd_api_key+" environment variable. You can get one at https://eodhd.com/")
	f.BoolVar(&c.inception, "inception", false, "ignore existing prices in ledger, and fetch all from inception date")
	f.Var(&c.tickers, "s", "security ticker to update (can be specified multiple times). If empty, all are updated.")
	f.BoolVar(&c.fetchForex, "forex", false, "fetch data for currency pairs")
	f.BoolVar(&c.fetchMSSI, "mssi", false, "fetch data for securities identified by MSSI")
	f.BoolVar(&c.fetchISIN, "isin", false, "fetch data for securities identified by ISIN")
	f.BoolVar(&c.fetchPrices, "prices", false, "fetch price data")
	f.BoolVar(&c.fetchSplits, "splits", false, "fetch split data")
	f.BoolVar(&c.fetchDividends, "dividends", false, "fetch dividend data")
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
	key := c.eodhdApiKey()
	if key == "" {
		fmt.Fprintf(os.Stderr, "Error: EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable\n")
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

	var options eodhd.FetchOptions
	if c.fetchForex {
		options |= eodhd.FetchForex
	}
	if c.fetchMSSI {
		options |= eodhd.FetchMSSI
	}
	if c.fetchISIN {
		options |= eodhd.FetchISIN
	}
	if c.fetchPrices {
		options |= eodhd.FetchPrices
	}
	if c.fetchSplits {
		options |= eodhd.FetchSplits
	}
	if c.fetchDividends {
		options |= eodhd.FetchDividends
	}

	for _, ledger := range ledgers {
		ledgerName := ledger.Name()
		fmt.Fprintf(os.Stderr, "Processing ledger %q...\n", ledgerName)
		_, updates, err := eodhd.Fetch(key, ledger, c.inception, options, c.tickers...)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: could not fetch from eodhd.com for ledger %q: %v\n", ledgerName, err)
			continue // Continue to the next ledger
		}

		// Update the ledger with the new market data.
		_, err = ledger.UpdateMarketData(updates...)
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
		if len(updates) == 0 {
			fmt.Printf("No updates for ledger %q.\n", ledgerName)
		}
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully fetched from eodhd.com and updated ledgers.\n")
	return subcommands.ExitSuccess
}
