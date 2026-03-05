package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/sg"
	"github.com/google/subcommands"
	"github.com/shopspring/decimal"
)

// sgFetchCmd implements the "sg fetch" command.
type sgFetchCmd struct {
	ledgerFile string
	inception  bool
}

func (*sgFetchCmd) Name() string     { return "fetch" }
func (*sgFetchCmd) Synopsis() string { return "fetches market data from Société Générale" }
func (*sgFetchCmd) Usage() string {
	return `sg fetch

Fetches holding positions details for all SG products and update ledger prices.
It only update prices for asset actually declared in the ledger.
`
}

func (c *sgFetchCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ledgerFile, "l", "", "ledger name to update. update all ledgers by default.")
	f.BoolVar(&c.inception, "inception", false, "ignore existing prices in ledger, and fetch all from inception date")
}

func (c *sgFetchCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	// Fetch available funds from SG
	funds, err := sg.FetchFunds()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: could not fetch SG funds list: %v\n", err)
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

		var updates []portfolio.Transaction
		var changes []struct {
			Date portfolio.Date
			ID   string
			New  decimal.Decimal
		}
		var count int

		snap := ledger.NewSnapshot(portfolio.Today())

		// Iterate over assets in the ledger to find matches
		for sec := range ledger.AllSecurities() {
			isin, err := sec.ID().ISIN()
			if err != nil {
				continue
			}
			if _, ok := funds[isin]; ok {
				// Determine the from Date.
				// The oldest possible value is inception date.
				from := ledger.InceptionDate(sec.Ticker())
				if !c.inception {
					// If we want incremental updates, start from the last known date + 1
					lastKnown := snap.LastMarketDataDate(sec.Ticker()).Add(1)
					// caveat: if there are no market data known (it's the case for new assets),
					// lastKnown will be the zero date, which is before inception date.
					// so we need to take the max of both.
					if lastKnown.After(from) {
						from = lastKnown
					}
				}

				fmt.Fprintf(os.Stderr, "Fetching history for %s...\n", isin)
				history, err := sg.FetchHistory(isin)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error fetching history for %s: %v\n", isin, err)
					continue
				}

				for _, point := range history {
					date, err := portfolio.ParseDate(point.Date)
					if err != nil {
						continue
					}
					if date.Before(from) {
						continue
					}
					val := decimal.NewFromFloat(point.Value)
					updates = append(updates, portfolio.NewUpdatePrice(
						date,
						sec.Ticker(),
						portfolio.M(val, sec.Currency()),
					))
					changes = append(changes, struct {
						Date portfolio.Date
						ID   string
						New  decimal.Decimal
					}{date, sec.Ticker(), val})
				}
				count++
			}
		}

		if len(updates) > 0 {
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
				fmt.Printf("%s %15s: %10s\n", c.Date, c.ID, c.New.StringFixed(4))
			}

			if summary.Total() > 0 {
				fmt.Fprintf(os.Stderr, "Finished processing ledger %q, %d updates applied from %d funds.\n", ledgerName, summary.Total(), count)
			} else {
				fmt.Fprintf(os.Stderr, "Finished processing ledger %q, no new updates from %d funds.\n", ledgerName, count)
			}
		} else {
			fmt.Fprintf(os.Stderr, "Finished processing ledger %q, no matching SG funds found.\n", ledgerName)
		}
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully fetched from SG and updated ledgers.\n")
	return subcommands.ExitSuccess
}
