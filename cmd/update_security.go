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
	start, end string  // for automatic mode
	id         string  // for manual mode
	date       string  // for manual mode
	price      float64 // for manual mode
}

func (*updateSecurityCmd) Name() string { return "update-security" }
func (*updateSecurityCmd) Synopsis() string {
	return "update security prices, either automatically or manually"
}
func (*updateSecurityCmd) Usage() string {
	return `pcs update-security [-start <date>] [-end <date>]
pcs update-security -id <security-id> -p <price> [-d <date>]

Updates security prices.
- In automatic mode (default), it fetches prices for all public securities.
- In manual mode (-id and -p flags), it sets a specific price for a security.
`
}

func (c *updateSecurityCmd) SetFlags(f *flag.FlagSet) {
	// Flags for automatic mode
	f.StringVar(&c.start, "start", date.New(2025, 01, 0).String(), "Set a specific start date to update from. See the user manual for supported date formats.")
	f.StringVar(&c.end, "end", date.Today().Add(-1).String(), "Set a specific end date. See the user manual for supported date formats.")

	// Flags for manual mode
	f.StringVar(&c.id, "id", "", "Unique security identifier (triggers manual mode)")
	f.Float64Var(&c.price, "p", 0, "Price to set for the security (triggers manual mode)")
	f.StringVar(&c.date, "d", date.Today().String(), "Date for the manual price update (optional, defaults to today). See the user manual for supported date formats.")
}

func (c *updateSecurityCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	// Determine mode based on flags
	isManualMode := c.id != "" && c.price != 0

	if isManualMode {
		// --- MANUAL MODE ---
		if f.NArg() != 0 {
			fmt.Fprintln(os.Stderr, "Error: no positional arguments are accepted in manual mode.")
			return subcommands.ExitUsageError
		}
		return c.executeManual(f)
	}

	// --- AUTOMATIC MODE ---
	if f.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "Error: no positional arguments are accepted in automatic mode.")
		return subcommands.ExitUsageError
	}
	return c.executeAutomatic(f)
}

func (c *updateSecurityCmd) executeAutomatic(_ *flag.FlagSet) subcommands.ExitStatus {
	start, err := date.Parse(c.start)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid start date:", err)
		return subcommands.ExitUsageError
	}
	end, err := date.Parse(c.end)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid end date:", err)
		return subcommands.ExitUsageError
	}

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

func (c *updateSecurityCmd) executeManual(_ *flag.FlagSet) subcommands.ExitStatus {
	day, err := date.Parse(c.date)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid date for manual update:", err)
		return subcommands.ExitUsageError
	}

	secID, err := portfolio.ParseID(c.id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error: invalid security ID:", err)
		return subcommands.ExitUsageError
	}

	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	if err := market.SetPrice(secID, day, c.price); err != nil {
		fmt.Fprintln(os.Stderr, "Error: failed to set price:", err)
		return subcommands.ExitFailure
	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Successfully set price for %s on %s to %.2f.\n", secID, day, c.price)
	return subcommands.ExitSuccess
}
