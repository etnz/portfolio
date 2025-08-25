package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

type updateCmd struct {
	start, end string
}

func (*updateCmd) Name() string { return "update" }
func (*updateCmd) Synopsis() string {
	return "update security prices from an external provider"
}
func (*updateCmd) Usage() string { return "pcs update\n" }
func (c *updateCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.start, "start", date.New(2025, 01, 0).String(), "set a specific start date to update from")
	f.StringVar(&c.end, "end", date.Today().Add(-1).String(), "set a specific end date")
}
func (c *updateCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		fmt.Println("no arguments expected")
		return subcommands.ExitUsageError
	}

	start, err := date.Parse(c.start)
	if err != nil {
		fmt.Println("invalid start date:", err)
		return subcommands.ExitUsageError
	}
	end, err := date.Parse(c.end)
	if err != nil {
		fmt.Println("invalid end date:", err)
		return subcommands.ExitUsageError
	}

	market, err := DecodeMarketData()
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	if err := market.Update(start, end); err != nil {
		fmt.Println("failed to update securities:", err)
		return subcommands.ExitFailure
	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
