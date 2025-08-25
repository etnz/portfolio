package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

type historyCmd struct {
	security string
	currency string
}

func (*historyCmd) Name() string     { return "history" }
func (*historyCmd) Synopsis() string { return "display asset value history" }
func (*historyCmd) Usage() string {
	return `history -s <security> | -c <currency>

  Displays the value of a single asset or cash account over time.
`
}

func (c *historyCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.security, "s", "", "security ticker to report on")
	f.StringVar(&c.currency, "c", "", "currency of cash account to report on")
}

func (c *historyCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if (c.security == "" && c.currency == "") || (c.security != "" && c.currency != "") {
		fmt.Fprintln(os.Stderr, "either -s or -c must be provided")
		return subcommands.ExitUsageError
	}

	as, err := DecodeAccountingSystem()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	var predicate func(portfolio.Transaction) bool
	if c.security != "" {
		predicate = portfolio.BySecurity(c.security)
	} else {
		predicate = portfolio.ByCurrency(c.currency)
	}

	if c.security != "" {
		fmt.Printf("Date\t\tPosition\tPrice\tValue\n")
	} else {
		fmt.Printf("Date\t\tValue\n")
	}

	for _, tx := range as.Ledger.Transactions(predicate) {
		on := tx.When()
		if c.security != "" {
			position := as.Ledger.Position(c.security, on)
			sec := as.Ledger.Get(c.security)
			price, ok := as.MarketData.PriceAsOf(sec.ID(), on)
			if !ok {
				fmt.Fprintf(os.Stderr, "error getting price for %s on %s\n", c.security, on)
				price = 0 // or some other default
			}
			value := position * price
			fmt.Printf("%s\t%.2f %.2f %.2f\n", on, position, price, value)

		} else {
			value := as.Ledger.CashBalance(c.currency, on)
			fmt.Printf("%s\t%.2f\n", on, value)
		}
	}

	return subcommands.ExitSuccess
}
