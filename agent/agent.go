package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/etnz/portfolio"
)

// Agent is the AI assistant that handles the chat session.
type Agent struct {
}

// New creates a new Agent.
func New(ledger *portfolio.Ledger) *Agent {
	return &Agent{}
}

// Run starts the interactive REPL session for the agent.
func (a *Agent) Run(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Welcome to  Assist. ctrl+C or 'bye' to exit.")

	for {
		fmt.Print("assist> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil // Clean exit on Ctrl+D
			}
			return err
		}

		if strings.TrimSpace(input) == "bye" {
			return nil
		}

		// For now, just echo the input.
		fmt.Printf("ECHO: %s", input)
	}
}
