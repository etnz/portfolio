package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/subcommands"
)

const amundiSessionFile = "pcs-amundi-session"

type headerFlags []string

func (h *headerFlags) String() string {
	return strings.Join(*h, ", ")
}

func (h *headerFlags) Set(value string) error {
	*h = append(*h, value)
	return nil
}

type amundiLoginCmd struct {
	headers headerFlags
	// Deprecated flags for curl compatibility
	curl string
	body string
}

func (*amundiLoginCmd) Name() string     { return "amundi-login" }
func (*amundiLoginCmd) Synopsis() string { return "stores Amundi session credentials from a curl command" }
func (*amundiLoginCmd) Usage() string {
	return `pcs amundi-login -H <header1> -H <header2> ...

Stores Amundi session credentials for use by the 'fetch amundi' command.
This command is designed to be user-friendly by accepting a pasted 'curl' command structure.
It extracts the necessary authentication headers and saves them to a temporary file.
`
}

func (c *amundiLoginCmd) SetFlags(f *flag.FlagSet) {
	f.Var(&c.headers, "H", "Header for the request (can be specified multiple times)")
	// Deprecated flags for curl compatibility
	f.StringVar(&c.curl, "curl", "", "ignored, for curl compatibility")
	f.StringVar(&c.body, "b", "", "ignored, for curl compatibility")
}

func (c *amundiLoginCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if len(c.headers) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one -H flag is required.")
		return subcommands.ExitUsageError
	}

	sessionPath := filepath.Join(os.TempDir(), amundiSessionFile)
	if err := os.WriteFile(sessionPath, []byte(strings.Join(c.headers, "\n")), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to save Amundi session: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Println("✅ Amundi session credentials successfully stored.")
	return subcommands.ExitSuccess
}
