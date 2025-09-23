package cmd

import (
	"context"
	"flag"

	"github.com/google/subcommands"
)

// amundiCmd is a container for amundi subcommands
type amundiCmd struct {
}

func (*amundiCmd) Name() string     { return "amundi" }
func (*amundiCmd) Synopsis() string { return "Amundi specific commands." }
func (*amundiCmd) Usage() string {
	return `amundi <subcommand> [args]

Commands:
  login - Authenticate with Amundi to create a session.
  fetch - Fetch latest prices for Amundi securities.
`
}

func (c *amundiCmd) SetFlags(f *flag.FlagSet) {}
func (c *amundiCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	commander := subcommands.NewCommander(f, "amundi")
	commander.Register(&amundiLoginCmd{}, "")
	commander.Register(&amundiFetchCmd{}, "")
	return commander.Execute(ctx, args...)
}
