package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

type updateSecurityCmd struct {
	id    string
	date  string
	price float64
	num   int
	den   int
}

func (*updateSecurityCmd) Name() string { return "update-security" }
func (*updateSecurityCmd) Synopsis() string {
	return "manually update a security's price or add a stock split"
}
func (*updateSecurityCmd) Usage() string {
	return `pcs update-security -id <security-id> [-p <price>] [-num <numerator>] [-den <denominator>] -d <date>

Manually modifies the data for a single security.
This command is used to set a specific price or record a stock split and does not fetch any data from the internet.
At least one of -p, -num, or -den must be provided.

  For example, to record a 2-for-1 stock split, use -num 2.
  For a 1-for-5 reverse split, use -den 5.
`
}

func (c *updateSecurityCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.id, "id", "", "Unique security identifier (required)")
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the update")
	f.Float64Var(&c.price, "p", 0, "Price to set for the security")
	f.IntVar(&c.num, "num", 1, "Split numerator")
	f.IntVar(&c.den, "den", 1, "Split denominator")
}

func (c *updateSecurityCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	priceSet := c.price != 0
	numSet := c.num != 1
	denSet := c.den != 1

	if c.id == "" {
		fmt.Fprintln(os.Stderr, "Error: -id flag is required.")
		return subcommands.ExitUsageError
	}

	if !priceSet && !numSet && !denSet {
		fmt.Fprintln(os.Stderr, "Error: at least one of -p, -num, or -den must be provided.")
		return subcommands.ExitUsageError
	}

	day, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid date:", err)
		return subcommands.ExitUsageError
	}

	secID, err := portfolio.ParseID(c.id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid security ID:", err)
		return subcommands.ExitUsageError
	}

	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading market data: %v\n", err)
		return subcommands.ExitFailure
	}

	if priceSet {
		if err := market.SetPrice(secID, day, c.price); err != nil {
			fmt.Fprintln(os.Stderr, "Error: failed to set price:", err)
			return subcommands.ExitFailure
		}
		fmt.Printf("Successfully set price for %s on %s to %.2f.\n", secID, day, c.price)
	}

	if numSet || denSet {
		split := portfolio.Split{
			Date:        day,
			Numerator:   c.num,
			Denominator: c.den,
		}
		market.AddSplit(secID, split)
		fmt.Printf("✅ Successfully added split for '%s' on %s.\n", c.id, c.date)
	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving market data: %v\n", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// executeAutomatic remains for later use by another command.
func (c *updateSecurityCmd) executeAutomatic(start, end date.Date) subcommands.ExitStatus {
	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	if err := market.Update(start, end); err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to automatically update securities:", err)
		return subcommands.ExitFailure
	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}
	fmt.Println("Successfully updated all public securities.")
	return subcommands.ExitSuccess
}
