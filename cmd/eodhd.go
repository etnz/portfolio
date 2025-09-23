package cmd

import (
	"context"
	"flag"

	"github.com/google/subcommands"
)

// eodhdCmd is the top-level command for EODHD-related operations.
type eodhdCmd struct{}

func (*eodhdCmd) Name() string     { return "eodhd" }
func (*eodhdCmd) Synopsis() string { return "EODHD provider specific commands" }
func (*eodhdCmd) Usage() string {
	return `eodhd <subcommand> <options>

EODHD provider specific commands.
`
}
func (c *eodhdCmd) SetFlags(f *flag.FlagSet) {}

func (c *eodhdCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	commander := subcommands.NewCommander(f, "eodhd")
	commander.Register(&eodhdFetchCmd{}, "")
	commander.Register(&eodhdSearchCmd{Cmd: &declareCmd{}}, "")
	return commander.Execute(ctx, args...)
}
