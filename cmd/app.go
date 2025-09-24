// Package cmd implements the CLI application to manage a portfolio.
package cmd

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/charmbracelet/glamour"
	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
	"github.com/shopspring/decimal"
)

// Register registers all the application's subcommands with the provided Commander.
// A main package will call Register() to set up the CLI.
func Register(c *subcommands.Commander) {
	c.Register(&amundiCmd{}, "providers")
	c.Register(&eodhdCmd{}, "providers")
	c.Register(&inseeCmd{}, "providers")

	c.Register(&initCmd{}, "transactions")
	c.Register(&buyCmd{}, "transactions")
	c.Register(&sellCmd{}, "transactions")
	c.Register(&dividendCmd{}, "transactions")
	c.Register(&depositCmd{}, "transactions")
	c.Register(&declareCmd{}, "transactions")
	c.Register(&withdrawCmd{}, "transactions")
	c.Register(&convertCmd{}, "transactions")
	c.Register(&accrueCmd{}, "transactions")
	c.Register(&priceCmd{}, "transactions")
	c.Register(&splitCmd{}, "transactions")

	c.Register(&fmtCmd{}, "tools")

	c.Register(&summaryCmd{}, "reports")
	c.Register(&holdingCmd{}, "reports")
	c.Register(&historyCmd{}, "reports")
	c.Register(&gainsCmd{}, "reports")
	c.Register(&logCmd{}, "reports")
	c.Register(&dailyCmd{}, "reports")
	c.Register(&weeklyCmd{}, "reports")
	c.Register(&monthlyCmd{}, "reports")
	c.Register(&quarterlyCmd{}, "reports")
	c.Register(&yearlyCmd{}, "reports")
	c.Register(&txCmd{}, "reports")
	c.Register(&reviewCmd{}, "reports")

	c.Register(&topicCmd{}, "documentation")

}

// As a CLI application, it has a very short-lived lifecycle, so it is ok to use global variables for flags.

var (
	ledgerFile      = flag.String("ledger-file", "transactions.jsonl", "Path to the ledger file containing transactions (JSONL format)")
	defaultCurrency = flag.String("default-currency", "EUR", "default currency")
	Verbose         = flag.Bool("v", false, "enable verbose logging")
	noRender        = flag.Bool("no-render", false, "disable markdown rendering in terminal output")
)

// DecodeLedger decodes the ledger from the application's default ledger file.
// If the file does not exist, it returns a new empty ledger.
func DecodeLedger() (*portfolio.Ledger, error) {
	f, err := os.Open(*ledgerFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			// If the file doesn't exist, it's an empty ledger.
			return portfolio.NewLedger(), nil
		}
		return nil, fmt.Errorf("could not open ledger file %q: %w", *ledgerFile, err)
	}
	defer f.Close()

	ledger, err := portfolio.DecodeLedger(f)
	if err != nil {
		return nil, fmt.Errorf("could not decode ledger file %q: %w", *ledgerFile, err)
	}
	return ledger, nil
}

// EncodeTransaction validates a transaction against the market data and existing
// ledger, then appends it to the ledger file.
func EncodeTransaction(tx portfolio.Transaction) (portfolio.Transaction, error) {
	ledger, err := DecodeLedger()
	if err != nil {
		return nil, fmt.Errorf("could not load ledger: %w", err)
	}

	tx, err = ledger.Validate(tx)
	if err != nil {
		return nil, err
	}

	// Open the file in append mode, creating it if it doesn't exist.
	f, err := os.OpenFile(*ledgerFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("error opening portfolio file %q: %w", *ledgerFile, err)
	}
	defer f.Close()

	if err := portfolio.EncodeTransaction(f, tx); err != nil {
		return nil, fmt.Errorf("error writing to portfolio file %q: %w", *ledgerFile, err)
	}
	return tx, nil
}

// printMarkdown renders a markdown string to stdout with appropriate styling.
// If styling fails for any reason (e.g., glamour error), it logs the
// error and falls back to printing the raw, un-styled markdown string.
func printMarkdown(md string) {
	if *noRender {
		fmt.Print(md)
		return
	}
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		log.Printf("Error creating markdown renderer: %v. Falling back to raw output.", err)
		fmt.Print(md)
		return
	}

	out, err := renderer.Render(md)
	if err != nil {
		log.Printf("Error rendering markdown: %v. Falling back to raw output.", err)
		fmt.Print(md)
		return
	}

	fmt.Print(out)
}

type decimalVar struct {
	f *decimal.Decimal
}

func (d decimalVar) String() string {
	if d.f == nil {
		return ""
	}
	return d.f.String()
}

func (d decimalVar) Set(s string) error {
	val, err := decimal.NewFromString(s)
	if err != nil {
		return err
	}
	*d.f = val
	return nil
}

func (d decimalVar) Type() string {
	return "decimal"
}

func DecimalVar(f *decimal.Decimal, def string) decimalVar {
	v := decimalVar{f: f}
	if err := v.Set(def); err != nil {
		panic("invalid default value for decimal var: " + err.Error())
	}
	return v
}

type quantityVar struct {
	f *portfolio.Quantity
}

func (d quantityVar) String() string {
	if d.f == nil {
		return ""
	}
	return d.f.String()
}

func (d quantityVar) Set(s string) error {
	dec, err := decimal.NewFromString(s)
	if err != nil {
		return err
	}
	val := portfolio.Q(dec)
	*d.f = val
	return nil
}

func QuantityVar(f *portfolio.Quantity, def string) quantityVar {
	v := quantityVar{f: f}
	if err := v.Set(def); err != nil {
		panic("invalid default value for quantity var: " + err.Error())
	}
	return v
}
