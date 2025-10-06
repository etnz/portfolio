package cmd

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/renderer"
	"github.com/google/subcommands"
)

type txCmd struct {
	period     string
	start      string
	date       string
	head       int
	tail       int
	ledgerFile string
}

func (*txCmd) Name() string     { return "tx" }
func (*txCmd) Synopsis() string { return "list all transactions in the ledger" }
func (*txCmd) Usage() string {
	return `pcs tx [-p <period> | -s <start_date>] [-d <end_date>] [-head <n>] [-tail <n>] [-l <ledger>]

  Lists transactions from the ledger, with options for filtering and limiting the output.
`
}

func (p *txCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&p.period, "p", "", "Predefined period (day, week, month, quarter, year).")
	f.StringVar(&p.start, "s", "", "The start date for a custom range. Overrides -p.")
	f.StringVar(&p.date, "d", "", "The end date for the range.")
	f.IntVar(&p.head, "head", 0, "Show only the first N transactions.")
	f.IntVar(&p.tail, "tail", 0, "Show only the last N transactions.")
	f.StringVar(&p.ledgerFile, "l", "", "Ledger to report on. Defaults to the only ledger if one exists.")
}

func (p *txCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if p.head > 0 && p.tail > 0 {
		fmt.Fprintln(os.Stderr, "Error: -head and -tail flags cannot be used together.")
		return subcommands.ExitUsageError
	}

	ledger, err := DecodeLedger(p.ledgerFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return subcommands.ExitFailure
	}

	var periodRange portfolio.Range
	// If no date range flags are provided, use the full range of the ledger.
	useFullRange := p.start == "" && p.date == "" && p.period == ""

	if !useFullRange {
		// Default end date to today if not provided
		endDateStr := p.date
		if endDateStr == "" {
			endDateStr = portfolio.Today().String()
		}
		endDate, err := portfolio.ParseDate(endDateStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing end date: %v\n", err)
			return subcommands.ExitFailure
		}

		if p.start != "" {
			startDate, err := portfolio.ParseDate(p.start)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing start date: %v\n", err)
				return subcommands.ExitFailure
			}
			periodRange = portfolio.NewRange(startDate, endDate)
		} else {
			period, err := portfolio.ParsePeriod(p.period)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error parsing period: %v\n", err)
				return subcommands.ExitFailure
			}
			periodRange = period.Range(endDate)
		}
	}

	var transactions []portfolio.Transaction
	for _, tx := range ledger.Transactions(portfolio.AcceptAll) {
		if useFullRange || periodRange.Contains(tx.When()) {
			transactions = append(transactions, tx)
		}
	}

	if p.head > 0 && len(transactions) > p.head {
		transactions = transactions[:p.head]
	}
	if p.tail > 0 && len(transactions) > p.tail {
		transactions = transactions[len(transactions)-p.tail:]
	}

	printMarkdown(renderer.Transactions(transactions))

	return subcommands.ExitSuccess
}
