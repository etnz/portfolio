package cmd

import (
	"context"
	"flag"

	"github.com/google/subcommands"
)

// inseeCmd is the top-level command for INSEE-related operations.
type inseeCmd struct{}

func (*inseeCmd) Name() string     { return "insee" }
func (*inseeCmd) Synopsis() string { return "INSEE provider specific commands" }
func (*inseeCmd) Usage() string {
	return `insee <subcommand> <options>

INSEE provider specific commands.
`
}
func (c *inseeCmd) SetFlags(f *flag.FlagSet) {}

func (c *inseeCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	commander := subcommands.NewCommander(f, "insee")
	commander.Register(&inseeFetchCmd{}, "")
	return commander.Execute(ctx, args...)
}
