package agent

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/genai"
)

// Agent is the AI assistant that handles the chat session.
type Agent struct {
	w           io.Writer
	r           *bufio.Reader
	Facilitator *Expert
	Experts     []*Expert
}

// New creates a new Agent. It initializes the Gemini client and the chat session.
//
// It takes a context, the portfolio ledger for answering user queries,
// an io.Writer for the agent's output (e.g., os.Stdout), and an io.Reader
// for user input (e.g., os.Stdin).
func New(w io.Writer, r io.Reader, experts ...*Expert) *Agent {
	return &Agent{
		w:           w,
		r:           bufio.NewReader(r),
		Experts:     experts,
		Facilitator: newFacilitator(experts...),
	}
}

func (a *Agent) Start(ctx context.Context, client *genai.Client) error {

	// At start create the Gemini all chats.
	for _, e := range a.Experts {
		if err := e.Start(ctx, client); err != nil {
			return err
		}
	}
	if err := a.Facilitator.Start(ctx, client); err != nil {
		return err
	}
	return nil
}

const prompt = "assist> "

// Run starts the interactive REPL session for the agent.
func (a *Agent) Run(ctx context.Context, client *genai.Client, prompts ...string) error {
	if a.Facilitator.chat == nil {
		if err := a.Start(ctx, client); err != nil {
			return err
		}
	}

	fmt.Fprintln(a.w, "Welcome to pcs financial assist. Type 'bye' to exit.")

	// REPL loop
	for {
		// Print the prompt
		fmt.Fprint(a.w, prompt)
		var input string

		// Flush prompts from the list and then ask for the user.
		if len(prompts) > 0 {
			input, prompts = prompts[0], prompts[1:]
			input = strings.TrimSpace(input)
			if input == "" {
				continue
			}
			fmt.Fprintln(a.w, input)
		} else {
			var err error
			input, err = a.r.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil // Clean exit on Ctrl+D
				}
				return err
			}
		}

		if strings.TrimSpace(input) == "bye" {
			return nil
		}

		content, err := a.Facilitator.Ask(ctx, &genai.Part{Text: input})
		if err != nil {
			return err
		}
		// TODO print markdown
		fmt.Fprintln(a.w, content.Parts[0].Text)
	}
}
