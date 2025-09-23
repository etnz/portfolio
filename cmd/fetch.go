package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/eodhd"
	"github.com/etnz/portfolio/insee"
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

	for cur := range ledger.Currencies() {
		if cur != ledger.Currency() {
			id, err := portfolio.NewCurrencyPair(cur, ledger.Currency())
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating currency pair for %s: %v\n", cur, err)
				return subcommands.ExitFailure
			}
			from := ledger.LastKnownMarketDataDate(id.String())
			requests[id] = portfolio.NewRange(from, portfolio.Today())
		}

	}

	for security := range ledger.AllSecurities() {
		id := security.ID()
		securities[id] = append(securities[id], security)

		var from portfolio.Date

		if c.inception { // --inception flag is set
			from = ledger.InceptionDate(security.Ticker())
		} else {
			// find the latest operation on this ticker
			lastOp := ledger.LastOperationDate(security.Ticker())
			if ledger.Position(lastOp, security.Ticker()).IsZero() {
				// No data is needed for this security.
				continue
			}

			// and the last market data date.
			from = ledger.LastKnownMarketDataDate(security.Ticker())
			if from.IsZero() {
				from = ledger.InceptionDate(security.Ticker())
			}
		}

		to := portfolio.Today()
		if from.After(to) {
			continue // Already up-to-date.
		}

		// If a range for this ID already exists, expand it to cover the new 'from' date if it's earlier.
		if existing, ok := requests[id]; !ok || from.Before(existing.From) {
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
		case "eodhd":
			for id, val := range requests {
				log.Printf("eodhd requested with %s from %s to %s\n", id, val.From, val.To)
			}
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
				log.Println("eodhd responded ", id, resp.Prices)
			}
		case "insee":
			inseeResponses, err := insee.Fetch(requests)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching from INSEE: %v\n", err)
				// Don't exit, other providers might succeed.
			}
			for id, resp := range inseeResponses {
				if len(resp.Dividends) == 0 && len(resp.Prices) == 0 && len(resp.Splits) == 0 {
					continue
				}
				allResponses[id] = resp
			}
		default:
			// External provider logic
			executableName := "pcs-fetch-" + provider
			path, err := exec.LookPath(executableName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: unsupported internal provider %q and could not find external provider executable %q in PATH. Skipping.\n", provider, executableName)
				continue
			}

			fmt.Fprintf(os.Stderr, "INFO: Using external provider %s\n", path)

			cmd := exec.Command(path)
			cmd.Stderr = os.Stderr // Pipe provider's stderr to ours for visibility

			// Pass configuration via environment variables
			absLedgerFile, _ := filepath.Abs(*ledgerFile)
			cmd.Env = os.Environ()
			cmd.Env = append(cmd.Env, fmt.Sprintf("PCS_LEDGER_FILE=%s", absLedgerFile))
			cmd.Env = append(cmd.Env, fmt.Sprintf("PCS_DEFAULT_CURRENCY=%s", ledger.Currency()))
			data, err := json.Marshal(requests)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error encoding requests: %v\n", err)
				continue
			}
			cmd.Stdin = bytes.NewReader(data)
			log.Println("sending request: ", string(data))

			output, err := cmd.Output()
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error running external provider %s: %v\n", executableName, err)
				continue
			}

			externalResponses := make(map[portfolio.ID]portfolio.ProviderResponse)
			if err := json.Unmarshal(output, &externalResponses); err != nil {
				fmt.Fprintln(os.Stderr, "received json: ", string(output))
				fmt.Fprintf(os.Stderr, "Error decoding external provider response: %v\n", err)
				continue
			}
			for id, resp := range externalResponses {
				allResponses[id] = resp
			}
		}
	}
	// if two providers are providing data for the same security current implementation will ignore the first one.

	// Append new data to the ledger.
	var newTxs []portfolio.Transaction
	// updatePrices will contain one UpdatePrice transaction per day
	updatePrices := make(map[portfolio.Date]portfolio.UpdatePrice)
	for id, resp := range allResponses {
		// Handle currency pairs separately as they don't have a user-defined ticker.
		if _, _, err := id.CurrencyPair(); err == nil {
			for date, price := range resp.Prices {
				upd, exist := updatePrices[date]
				if !exist {
					upd = portfolio.NewUpdatePrices(date, make(map[string]decimal.Decimal))
				}
				upd.Prices[id.String()] = decimal.NewFromFloat(price)
				updatePrices[date] = upd
			}
			continue // Move to the next response.
		}

		// Do not trust the provider. Only process data for securities that were
		// actually requested.
		reqRange, requested := requests[id]
		if !requested {
			continue
		}

		// Create transactions for each ticker associated with this ID.
		for _, sec := range securities[id] {
			for date, price := range resp.Prices {
				// This is a difficult problem. I need to know at least
				// one price before I declare or actually start using an asset.
				// So if I am asking for a value on a particular (say reqRange.From)
				// And there are not value for that day, I should be allowed to
				// let an earlier value in.
				// Remove the following block would do that.
				// but then, the ledger might become invalid (updatePrice of unknown asset).
				// So there is no simple solution, other than declaring the asset early enough so that
				// it contains enough update prices.
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
