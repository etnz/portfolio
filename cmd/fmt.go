package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type fmtCmd struct {
	outputFile string
}

func (*fmtCmd) Name() string { return "fmt" }
func (*fmtCmd) Synopsis() string {
	return "validates and formats the ledger file into a canonical form"
}
func (*fmtCmd) Usage() string {
	return `pcs fmt [-o <file_path>]

  Validates and formats the ledger file. This command reads all transactions,
  validates them, applies available quick-fixes (like resolving "sell all"),
  sorts them by date, and writes them back in a canonical JSONL format.
  If -o is not specified, it overwrites the default ledger file. Use -o - to write to stdout.

Usage Examples:
# Writes to the default ledger file.
$ pcs fmt

# Writes the output to /tmp/my-custom-ledger.txt.
$ pcs fmt -o /tmp/my-custom-ledger.txt

# Writes output to stdout, which is then piped to the 'less' command.
$ pcs fmt -o - | less
`
}

func (p *fmtCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.outputFile, "o", "", "Output file path. Use '-' for stdout.")
}

func (p *fmtCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	log.Printf("input file=%s", *ledgerFile)

	// We need to decode with validation the
	file, err := os.Open(*ledgerFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "No ledger file found at %q\n", *ledgerFile)
			return subcommands.ExitFailure
		}
		fmt.Fprintf(os.Stderr, "could not open ledger file %q: %v", *ledgerFile, err)
		return subcommands.ExitFailure
	}
	ledger, err := portfolio.DecodeValidateLedger(file)
	file.Close()
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
