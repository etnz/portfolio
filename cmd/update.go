package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type updateCmd struct{}

func (*updateCmd) Name() string { return "update" }
func (*updateCmd) Synopsis() string {
	return "update security prices in from eodhd.com provider"
}
func (*updateCmd) Usage() string              { return "pcs update\n" }
func (c *updateCmd) SetFlags(f *flag.FlagSet) {}
func (c *updateCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 0 {
		fmt.Println("no arguments expected")
		return subcommands.ExitUsageError
	}

	db, err := OpenSecurities()
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	if err := db.Update(); err != nil {
		fmt.Println("failed to update securities:", err)
		return subcommands.ExitFailure
	}

	if err := CloseSecurities(db); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
