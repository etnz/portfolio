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
	"github.com/shopspring/decimal"
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
				if len(resp.Dividends) == 0 && len(resp.Prices) == 0 && len(resp.Splits) == 0 {
					continue
				}
				allResponses[id] = resp
			}
		case "eodhd":
			eodhdResponses, err := eodhd.Fetch(requests)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching from EODHD: %v\n", err)
				// Don't exit, other providers might succeed.
			}
			for id, resp := range eodhdResponses {
				if len(resp.Dividends) == 0 && len(resp.Prices) == 0 && len(resp.Splits) == 0 {
					continue
				}
				allResponses[id] = resp
			}
		default:
			fmt.Fprintf(os.Stderr, "Warning: unsupported provider %q, skipping.\n", provider)
		}
	}
	// if two providers are providing data for the same security current implementation will ignore the first one.

	// Append new data to the ledger.
	var newTxs []portfolio.Transaction
	// updatePrices will contain one UpdatePrice transaction per day
	updatePrices := make(map[portfolio.Date]portfolio.UpdatePrice)
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

				upd, exist := updatePrices[date]
				if !exist {
					upd = portfolio.NewUpdatePrices(date, make(map[string]decimal.Decimal))
				}
				upd.Prices[sec.Ticker()] = decimal.NewFromFloat(price)
				updatePrices[date] = upd

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
		// now append all the update prices (one per day at once)
		for _, upd := range updatePrices {
			newTxs = append(newTxs, upd)
		}
	}

	if len(newTxs) == 0 {
		fmt.Println("No new market data found.")
		return subcommands.ExitSuccess
	}

	// This is a mess. it will append market data even for positions not held.
	// Then we call Clean() to remove them (incorrectly btw).
	// we are loosing track of what number of actual updates.
	// SO AppendOrUpdate should not append useless data market, and therefore should compute holdings
	// on each date.

	// 4. Apply updates to the ledger and write it back
	upd, err := ledger.UpdateMarketData(newTxs...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating ledger: %v\n", err)
		return subcommands.ExitFailure
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

	if upd.Total() == 0 {
		fmt.Println("✅ Successfully fetch updates, but there are no new data points.")
		return subcommands.ExitSuccess
	}

	fmt.Printf("✅ Successfully updated %d data points.\n", upd.Total())
	if upd.NewSplits() > 0 {
		fmt.Printf(" * %d new splits.\n", upd.NewSplits())
	}
	if upd.UpdatedSplits() > 0 {
		fmt.Printf(" * %d updated splits.\n", upd.UpdatedSplits())
	}
	if upd.AddedDividends() > 0 {
		fmt.Printf(" * %d new dividends.\n", upd.AddedDividends())
	}
	if upd.UpdatedDividends() > 0 {
		fmt.Printf(" * %d updated dividends.\n", upd.UpdatedDividends())
	}
	if upd.AddedPrices() > 0 {
		fmt.Printf(" * %d new prices.\n", upd.AddedPrices())
	}
	if upd.UpdatedPrices() > 0 {
		fmt.Printf(" * %d updated prices.\n", upd.UpdatedPrices())
	}

	return subcommands.ExitSuccess
}
