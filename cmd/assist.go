package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/etnz/portfolio/agent"
	"github.com/google/subcommands"
)

// AssistCmd is the subcommand for the AI assistant.
type AssistCmd struct{}

// Name returns the name of the command.
func (*AssistCmd) Name() string { return "assist" }

// Synopsis returns a short-one line synopsis of the command.
func (*AssistCmd) Synopsis() string { return "Start an interactive session with the AI assistant." }

// Usage returns a long-form usage string.
func (*AssistCmd) Usage() string {
	return `assist:
  Start an interactive session with the AI assistant.
`
}

// SetFlags sets the flags for the command.
func (*AssistCmd) SetFlags(_ *flag.FlagSet) {}

// Execute executes the command.
func (c *AssistCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Println("Error loading ledger:", err)
		return subcommands.ExitFailure
	}

	ag := agent.New(ledger)
	if err := ag.Run(ctx); err != nil {
		fmt.Println("Agent failed:", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
