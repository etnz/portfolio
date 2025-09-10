package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type fetchSecurityCmd struct {
	id     string
	prices bool
	splits bool
	from   string
	to     string
}

func yesterday() portfolio.Date {
	return portfolio.Today().Add(-1)
}
func lastYear() portfolio.Date {
	return portfolio.Today().Add(-365)
}

func (*fetchSecurityCmd) Name() string { return "fetch-security" }
func (*fetchSecurityCmd) Synopsis() string {
	return "fetches and updates market data from external providers"
}
func (*fetchSecurityCmd) Usage() string {
	return `pcs fetch-security [-id <security-id>] [-prices] [-splits] [-from <date>] [-to <date>]

Fetches and updates market data from external providers.
By default, it fetches all available data (prices and splits) for all securities.
Flags can be used to limit the scope of the fetch.
`
}

func (c *fetchSecurityCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.id, "id", "", "Fetch data only for the specified security ID")
	f.BoolVar(&c.prices, "prices", false, "Only fetch price history")
	f.BoolVar(&c.splits, "splits", false, "Only fetch split history")
	f.StringVar(&c.from, "from", lastYear().String(), "Start date for fetching historical data (YYYY-MM-DD)")
	f.StringVar(&c.to, "to", yesterday().String(), "End date for fetching historical data (YYYY-MM-DD)")
}

func (c *fetchSecurityCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	start, err := portfolio.ParseDate(c.from)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid from date: %v\n", err)
		return subcommands.ExitUsageError
	}

	end, err := portfolio.ParseDate(c.to)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: invalid to date: %v\n", err)
		return subcommands.ExitUsageError
	}

	// If no data type flag is specified, fetch all types.
	if !c.prices && !c.splits {
		c.prices = true
		c.splits = true
	}

	if c.id != "" {
		// TODO: implement fetching for a single security
		fmt.Fprintln(os.Stderr, "Error: fetching for a single security is not yet implemented")
		return subcommands.ExitFailure
	}

	if c.prices {
		if err := market.UpdatePrices(start, end); err != nil {
			fmt.Fprintln(os.Stderr, "Error: failed to automatically update securities:", err)
			// continue to fetch splits
		}
	}

	if c.splits {
		if err := market.UpdateSplits(); err != nil {
			fmt.Fprintln(os.Stderr, "Error: failed to update splits:", err)
			return subcommands.ExitFailure
		}
	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}
	fmt.Println("Successfully updated all public securities.")
	return subcommands.ExitSuccess
}
