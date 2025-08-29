package cmd

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type formatLedgerCmd struct {
	outputFile string
}

func (*formatLedgerCmd) Name() string     { return "format-ledger" }
func (*formatLedgerCmd) Synopsis() string { return "formats the ledger file into a canonical form" }
func (*formatLedgerCmd) Usage() string {
	return `pcs format-ledger [-o <file_path>]

  Formats the ledger file into a canonical form, sorting transactions by date
  and JSON keys alphabetically. If -o is not specified, it overwrites the
  default ledger file. Use -o - to write to stdout.

Usage Examples:
# Writes to the default ledger file.
$ pcs format-ledger

# Writes the output to /tmp/my-custom-ledger.txt.
$ pcs format-ledger -o /tmp/my-custom-ledger.txt

# Writes output to stdout, which is then piped to the 'less' command.
$ pcs format-ledger -o - | less
`
}

func (p *formatLedgerCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.outputFile, "o", "", "Output file path. Use '-' for stdout.")
}

func (p *formatLedgerCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	log.Printf("input file=%s", *ledgerFile)
	ledger, err := DecodeLedger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	// Determine output
	var writer io.Writer
	var confirmationMessage string

	switch p.outputFile {
	case "":
		// Default behavior: write to ledgerFile
		outputPath := *ledgerFile
		log.Printf("output file=%s", outputPath)
		file, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening default ledger file %q for writing: %v\n", outputPath, err)
			return subcommands.ExitFailure
		}
		defer file.Close()
		writer = file
		confirmationMessage = fmt.Sprintf("Default ledger file '%s' has been formatted.\n", outputPath)
	case "-":
		// Write to stdout
		writer = os.Stdout
		log.Printf("output to stdout")

	default:
		// Write to specified output file
		outputPath := p.outputFile
		log.Printf("output file=%s", outputPath)
		file, err := os.Create(outputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening output file %q for writing: %v\n", outputPath, err)
			return subcommands.ExitFailure
		}
		defer file.Close()
		writer = file
		confirmationMessage = fmt.Sprintf("Ledger file '%s' has been formatted.\n", outputPath)
	}

	// 2. Write the ledger
	err = portfolio.EncodeLedger(writer, ledger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Print(confirmationMessage)
	return subcommands.ExitSuccess
}
