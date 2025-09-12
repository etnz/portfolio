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

type fetchCmd struct {
}

func (*fetchCmd) Name() string     { return "fetch" }
func (*fetchCmd) Synopsis() string { return "fetches market data from an external provider" }
func (*fetchCmd) Usage() string {
	return `pcs fetch <provider...>

Fetches market data (prices, splits, dividends) from an external provider
and appends the new data to the ledger as transactions.

It analyzes the ledger to determine the required date range for fetching.
At least one provider must be specified.

Supported providers:
  - amundi: Fetches data for Amundi funds. Requires 'pcs amundi-login' first.
`
}

func (c *fetchCmd) SetFlags(f *flag.FlagSet) {}

func (c *fetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not load ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	// 1. Build fetch requests based on ledger analysis
	requests := make(map[portfolio.ID]portfolio.Range)
	securities := make(map[portfolio.ID][]portfolio.Security)
	for security := range ledger.AllSecurities() {
		id := security.ID()
		securities[id] = append(securities[id], security)

		from, ok := ledger.LastKnownMarketDataDate(security.Ticker())
		if !ok {
			// No market data yet, use the security's inception date.
			from, ok = ledger.InceptionDate(security.Ticker())
			if !ok {
				// No transactions for this specific ticker, skip.
				continue
			}
		} else {
			// Fetch from the day after the last known data point.
			from = from.Add(1)
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

	// 2. Call the provider(s)
	providers := f.Args()
	if len(providers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one provider must be specified.")
		f.Usage()
		return subcommands.ExitUsageError
	}

	allResponses := make(map[portfolio.ID]portfolio.ProviderResponse)

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
				allResponses[id] = resp
			}
		default:
			fmt.Fprintf(os.Stderr, "Warning: unsupported provider %q, skipping.\n", provider)
		}
	}

	// 3. Append new data to the ledger
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
			// TODO(#74): Re-enable dividend fetching once market data is fully decoupled from validation.
			// The validation for a Dividend transaction requires calculating the total amount
			// from a per-share value. This calculation depends on the exact position on the
			// dividend date, which can be affected by stock splits.
			// Currently, split data is still partially tied to the market.jsonl file,
			// which is not available during the ledger-only validation phase of the `fetch`
			// command. This creates a dependency issue.
			//
			// for date, dividend := range resp.Dividends {
			// 	if !reqRange.Contains(date) {
			// 		continue
			// 	}
			// 	dps := portfolio.M(dividend.Amount, sec.Currency())
			// 	// Create a dividend transaction with a zero total amount. The validation step will calculate it.
			// 	tx := portfolio.NewDividend(date, "fetched from provider", sec.Ticker(), portfolio.M(0, sec.Currency()))
			// 	tx.DividendPerShare = dps
			// 	newTxs = append(newTxs, tx)
			// }
		}
	}

	if len(newTxs) == 0 {
		fmt.Println("No new market data found.")
		return subcommands.ExitSuccess
	}

	// 4. Apply updates to the ledger and write it back
	ledger.AppendOrUpdate(newTxs...)

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
