package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/amundi"
	"github.com/etnz/portfolio/eodhd"
	"github.com/etnz/portfolio/insee"
	"github.com/etnz/portfolio/sg"
	"github.com/etnz/portfolio/yahoo"
	"github.com/google/subcommands"
)

// Fetcher is the interface that must be implemented by a market data provider.
// It fetches market data for a single security within a given date range.
type Fetcher func(ctx context.Context, security portfolio.Security, from, to portfolio.Date) ([]portfolio.Transaction, error)

var fetcherRegistry = make(map[string]Fetcher)

// RegisterFetcher registers a fetcher for a specific protocol.
func RegisterFetcher(protocol string, f Fetcher) {
	fetcherRegistry[protocol] = f
}

func init() {
	RegisterFetcher("yahoo", yahoo.Fetch)
	RegisterFetcher("insee", insee.Fetch)
	RegisterFetcher("amundi", amundi.Fetch)
	RegisterFetcher("eodhd", eodhd.FetchSecurity)
	RegisterFetcher("sg", sg.Fetch)
}

type stringList []string

func (s *stringList) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *stringList) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type fetchCmd struct {
	ledgerFile string
	inception  bool
	providers  stringList
}

func (*fetchCmd) Name() string { return "fetch" }
func (*fetchCmd) Synopsis() string {
	return "fetches market data for all securities with a declared fetcher"
}
func (*fetchCmd) Usage() string {
	return `pcs fetch [-l <ledger>] [-inception] [-f <provider>]

Fetches market data for all securities in the ledger that have a 'fetcher' URI declared.
It uses the protocol specified in the URI (e.g., "yahoo://") to select the appropriate provider.
`
}

func (c *fetchCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.ledgerFile, "l", "", "Ledger to update. Updates all ledgers by default.")
	f.BoolVar(&c.inception, "inception", false, "Fetch data from inception date, ignoring existing history.")
	f.Var(&c.providers, "f", "Limit fetch to specific providers (e.g. -f yahoo). Can be repeated.")
}

func (c *fetchCmd) Execute(ctx context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	ledgers, err := DecodeLedgers(c.ledgerFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading ledgers: %v\n", err)
		return subcommands.ExitFailure
	}

	for _, ledger := range ledgers {
		if err := c.processLedger(ctx, ledger); err != nil {
			fmt.Fprintf(os.Stderr, "Error processing ledger %q: %v\n", ledger.Name(), err)
		}
	}
	return subcommands.ExitSuccess
}

func (c *fetchCmd) processLedger(ctx context.Context, ledger *portfolio.Ledger) error {
	log.Printf("Processing ledger %q...", ledger.Name())

	var allUpdates []portfolio.Transaction

	snap := ledger.NewSnapshot(portfolio.Today())

	for sec := range ledger.AllSecurities() {
		// Determine date range
		var to portfolio.Date
		if sec.ID().IsCurrencyPair() {
			// For currencies, the EOD price is for the previous day.
			to = portfolio.Today().Add(-1)
		} else {
			to = portfolio.Today()
			// Optimization: if the asset is no longer held, we don't need to fetch data up to today.
			// We can stop at the last operation date (e.g. the date of the sale).
			lastOp := ledger.LastOperationDate(sec.Ticker())
			if !lastOp.IsZero() && ledger.Position(lastOp, sec.Ticker()).IsZero() {
				to = lastOp.Add(-1)
			}
		}

		// Determine the from Date.
		from := ledger.InceptionDate(sec.Ticker())
		if !c.inception {
			// If we want incremental updates, start from the last known date + 1
			lastKnown := snap.LastMarketDataDate(sec.Ticker()).Add(1)
			if lastKnown.After(from) {
				from = lastKnown
			}
		}

		if from.After(to) {
			continue
		}

		// An update is required, try to load the fetcher.
		uriStr := sec.Fetcher()
		if uriStr == "" {

			log.Printf("Security %s (%s) has no fetcher URI for [%v, %v]", sec.Ticker(), sec.ID(), from, to)
			continue
		}

		u, err := url.Parse(uriStr)
		if err != nil {
			log.Printf("Invalid fetcher URI %q for %s: %v", uriStr, sec.Ticker(), err)
			continue
		}

		protocol := u.Scheme

		if len(c.providers) > 0 {
			found := false
			for _, p := range c.providers {
				if p == protocol {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		fetcher, ok := fetcherRegistry[protocol]
		if !ok {
			log.Printf("No fetcher registered for protocol %q (security %s)", protocol, sec.Ticker())
			continue
		}

		fmt.Fprintf(os.Stderr, "Fetching %s data for %s (%s to %s)...\n", protocol, sec.Ticker(), from, to)
		updates, err := fetcher(ctx, sec, from, to)
		if err != nil {
			log.Printf("Error fetching data for %s: %v", sec.Ticker(), err)
			continue
		}

		allUpdates = append(allUpdates, updates...)
	}

	if len(allUpdates) > 0 {
		summary, err := ledger.UpdateMarketData(allUpdates...)
		if err != nil {
			return fmt.Errorf("failed to update market data: %w", err)
		}

		if err := portfolio.SaveLedger(PortfolioPath(), ledger); err != nil {
			return fmt.Errorf("failed to save ledger: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Updated ledger %q: %d prices, %d splits, %d dividends.\n",
			ledger.Name(), summary.AddedPrices()+summary.UpdatedPrices(), summary.NewSplits()+summary.UpdatedSplits(), summary.AddedDividends()+summary.UpdatedDividends())
	} else {
		fmt.Fprintf(os.Stderr, "No updates found for ledger %q.\n", ledger.Name())
	}

	return nil
}
