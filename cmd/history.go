package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/google/subcommands"
)

// historyCmd holds the flags for the 'history' subcommand.
type historyCmd struct {
	security string
	currency string
}

func (*historyCmd) Name() string     { return "history" }
func (*historyCmd) Synopsis() string { return "display asset value history" }
func (*historyCmd) Usage() string {
	return `pcs history -s <security> | -c <currency>

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
		balance, err := as.Balance(on)
		if c.security != "" {
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error calculating balance: %v\n", err)
				return subcommands.ExitFailure
			}
			position := balance.Position(c.security)
			price := balance.Price(c.security)
			value := balance.MarketValue(c.security)
			fmt.Printf("%s    %-10s %-10s %-10s\n", on, position, price, value)
		} else {
			value := balance.Cash(c.currency)
			fmt.Printf("%s    %-10s\n", on, value)
		}
	}

	return subcommands.ExitSuccess
}
