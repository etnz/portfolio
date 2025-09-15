package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/amundi"
	"github.com/etnz/portfolio/eodhd"
	"github.com/google/subcommands"
)

type fetchCmd struct {
	inception bool
}

func (*fetchCmd) Name() string     { return "fetch" }
func (*fetchCmd) Synopsis() string { return "fetches market data from an external provider" }
func (*fetchCmd) Usage() string {
	return `pcs fetch <provider...>

Fetches market data (prices, splits, dividends) from an external provider
and appends the new data to the ledger as transactions.

It analyzes the ledger to determine the required date range for fetching.
At least one provider must be specified.

The --inception flag can be used to ignore the last known data point and
force a full refresh from the security's first transaction date.

Supported providers:
  - amundi: Fetches data for Amundi funds. Requires 'pcs amundi-login' first.
  - eodhd:  Fetches data from EOD Historical Data. Requires an API key
            set via the --eodhd-api-key flag or the EODHD_API_KEY
            environment variable.

Note: If a provider is not specified, 'eodhd' will be used by default
if an API key is available.
`
}

func (c *fetchCmd) SetFlags(f *flag.FlagSet) {
	f.BoolVar(&c.inception, "inception", false, "Force fetching data from the security's inception date.")
}

func (c *fetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	// Build fetch requests based on ledger analysis.
	requests := make(map[portfolio.ID]portfolio.Range)
	securities := make(map[portfolio.ID][]portfolio.Security)
	for security := range ledger.AllSecurities() {
		id := security.ID()
		securities[id] = append(securities[id], security)

		var from portfolio.Date
		var ok bool

		if c.inception { // --inception flag is set
			from, ok = ledger.InceptionDate(security.Ticker())
		} else {
			// Default behavior: start from the last known data point.
			from, ok = ledger.LastKnownMarketDataDate(security.Ticker())
			if !ok {
				// Fallback to inception date if no market data is known.
				from, ok = ledger.InceptionDate(security.Ticker())
			}
		}

		if !ok { // If no valid start date could be found, skip this security.
			continue
		}

		to := portfolio.Today()
		if from.After(to) {
			continue // Already up-to-date.
		}

		// If a range for this ID already exists, expand it to cover the new 'from' date if it's earlier.
		if existing, ok := requests[id]; !ok {
			requests[id] = portfolio.NewRange(from, to)
		} else if from.Before(existing.From) {
			// expand the range
			requests[id] = portfolio.NewRange(from, to)
		}
	}

	if len(requests) == 0 {
		fmt.Println("All securities are up-to-date. Nothing to fetch.")
		return subcommands.ExitSuccess
	}

	// Call the provider(s).
	providers := f.Args()
	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one provider must be specified.")
		f.Usage()
		return subcommands.ExitUsageError
	}

	allResponses := make(map[portfolio.ID]portfolio.ProviderResponse)

	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one provider must be specified, and no default provider could be set.")
		f.Usage()
		return subcommands.ExitUsageError
	}

	for _, provider := range providers {
		switch provider {
		case "amundi":
			amundiResponses, err := amundi.Fetch(requests)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching from Amundi: %v\n", err)
				// Don't exit, other providers might succeed.
			}
			// In the future, we would merge responses carefully.
			for id, resp := range amundiResponses {
				// TODO: ignore empty responses.
				allResponses[id] = resp
			}
		case "eodhd":
			eodhdResponses, err := eodhd.Fetch(requests)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching from EODHD: %v\n", err)
				// Don't exit, other providers might succeed.
			}
			for id, resp := range eodhdResponses {
				// TODO: ignore empty responses or merge with other provider responses.
				allResponses[id] = resp
			}
		default:
			fmt.Fprintf(os.Stderr, "Warning: unsupported provider %q, skipping.\n", provider)
		}
	}

	// Append new data to the ledger.
	var newTxs []portfolio.Transaction
	for id, resp := range allResponses {
		// Do not trust the provider. Only process data for securities that were
		// actually requested.
		reqRange, requested := requests[id]
		if !requested {
			continue
		}

		// Create transactions for each ticker associated with this ID.
		for _, sec := range securities[id] {
			for date, price := range resp.Prices {
				if !reqRange.Contains(date) {
					continue // Only create transactions for the requested date range.
				}
				tx := portfolio.NewUpdatePrice(date, sec.Ticker(), portfolio.M(price, sec.Currency()))
				newTxs = append(newTxs, tx)
			}
			for date, split := range resp.Splits {
				if !reqRange.Contains(date) {
					continue
				}
				tx := portfolio.NewSplit(date, sec.Ticker(), split.Numerator, split.Denominator)
				newTxs = append(newTxs, tx)
			}
			for date, dividend := range resp.Dividends {
				if !reqRange.Contains(date) {
					continue
				}
				// Create a dividend transaction with a zero total amount. The validation
				// step, which runs when the ledger is loaded, will calculate the
				// total amount based on the position at that date.
				dps := portfolio.M(dividend.Amount, sec.Currency())
				tx := portfolio.NewDividend(date, "fetched from provider", sec.Ticker(), dps)
				newTxs = append(newTxs, tx)
			}
		}
	}

	if len(newTxs) == 0 {
		fmt.Println("No new market data found.")
		return subcommands.ExitSuccess
	}

	// 4. Apply updates to the ledger and write it back
	ledger.AppendOrUpdate(newTxs...)

	// now we have imported probably too much market data, indeed we have market data about assets in period were
	// we don't own them. We need to clean them up.
	// We do a quick journal scan, to compute only positions, and on the fly build the list of spurious transactions.
	if err := ledger.Clean(); err != nil {
		fmt.Fprintf(os.Stderr, "Error cleaning ledger: %v\n", err)
	}

	file, err := os.Create(*ledgerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening ledger file for writing: %v\n", err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	if err := portfolio.EncodeLedger(file, ledger); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing updated ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("✅ Successfully fetched and appended %d new market data points to the ledger.\n", len(newTxs))
	return subcommands.ExitSuccess
}
