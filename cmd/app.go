// Package cmd implements the CLI application to manage a portfolio.
package cmd

import (
	"flag"
	"fmt"
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
	c.Register(&AssistCmd{}, "tools")

	c.Register(&summaryCmd{}, "reports")
	c.Register(&holdingCmd{}, "reports")
	c.Register(&historyCmd{}, "reports")
	c.Register(&txCmd{}, "reports")
	c.Register(&reviewCmd{}, "reports")

	c.Register(&topicCmd{}, "documentation")

}

// As a CLI application, it has a very short-lived lifecycle, so it is ok to use global variables for flags.

var (
	defaultCurrency = flag.String("default-currency", "EUR", "default currency")
	Verbose         = flag.Bool("v", false, "enable verbose logging")
	noRender        = flag.Bool("no-render", false, "disable markdown rendering in terminal output")
	portfolioPath   = flag.String("portfolio", "", "Path to the portfolio directory (overrides PORTFOLIO_PATH env var)")
)

// PortfolioPath resolves the path to the portfolio directory.
// It follows this order of precedence:
// 1. --portfolio flag
// 2. PORTFOLIO_PATH environment variable
// 3. Current working directory (".")
func PortfolioPath() string {
	if *portfolioPath != "" {
		return *portfolioPath
	}
	if envPath := os.Getenv("PORTFOLIO_PATH"); envPath != "" {
		return envPath
	}
	return "."
}

// DecodeLedger decodes the ledger from the application's default ledger file.
// If the file does not exist, it returns a new empty ledger.
func DecodeLedger(query string) (*portfolio.Ledger, error) {
	path := PortfolioPath()
	return portfolio.FindLedger(path, query)
}

// DecodeLedgers decodes all ledgers from the portfolio path.
func DecodeLedgers(query string) ([]*portfolio.Ledger, error) {
	path := PortfolioPath()
	return portfolio.FindLedgers(path, query)
}

// EncodeTransaction validates a transaction against the market data and existing
// ledger, then appends it to the ledger file.
func EncodeTransaction(ledger *portfolio.Ledger, tx portfolio.Transaction) (portfolio.Transaction, error) {
	validatedTx, err := ledger.Validate(tx)
	if err != nil {
		return nil, err
	}

	if err := ledger.Append(validatedTx); err != nil {
		return nil, fmt.Errorf("could not append transaction: %w", err)
	}

	if err := portfolio.SaveLedger(PortfolioPath(), ledger); err != nil {
		return nil, fmt.Errorf("could not save ledger: %w", err)
	}

	return validatedTx, nil
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
