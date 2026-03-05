package cmd

import (
	"context"
	"flag"

	"github.com/google/subcommands"
)

// sgCmd is the top-level command for SG-related operations.
type sgCmd struct{}

func (*sgCmd) Name() string     { return "sg" }
func (*sgCmd) Synopsis() string { return "Société Générale provider specific commands" }
func (*sgCmd) Usage() string {
	return `sg <subcommand> <options>

Société Générale provider specific commands.
`
}
func (c *sgCmd) SetFlags(f *flag.FlagSet) {}

func (c *sgCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	commander := subcommands.NewCommander(f, "sg")
	commander.Register(&sgFetchCmd{}, "")
	return commander.Execute(ctx, args...)
}
