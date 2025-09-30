package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/etnz/portfolio/agent"
	"github.com/google/subcommands"
	"google.golang.org/genai"
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
	var err error
	initialPrompt := ""
	if f.NArg() > 0 {
		initialPrompt = strings.Join(f.Args(), " ")

	}

	// ledger, err := DecodeLedger()
	// if err != nil {
	// 	fmt.Println("Error loading ledger:", err)
	// 	return subcommands.ExitFailure
	// }

	client, err := genai.NewClient(ctx, nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing Gemini's client:", err)
		return subcommands.ExitFailure
	}

	trader := agent.NewTrader()
	accountant := agent.NewAccountant()
	a := agent.New(os.Stdout, os.Stdin, trader, accountant)

	if err := a.Run(ctx, client, initialPrompt); err != nil {
		fmt.Fprintln(os.Stderr, "Agent failed:", err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
