package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
	"github.com/shopspring/decimal"
)

// 

func main() {
	// The migrate tool needs its own set of flags, independent of the main pcs tool.
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	commander := subcommands.NewCommander(flag.CommandLine, "migrate")
	commander.Register(&marketCmd{}, "")
	commander.Register(&ledgerCmd{}, "")
	commander.Register(&checkCmd{}, "")
	flag.Parse()
	os.Exit(int(commander.Execute(context.Background())))
}

// --- marketCmd ---

type marketCmd struct {
	in  string
	out string
}

func (*marketCmd) Name() string { return "market" }
func (*marketCmd) Synopsis() string {
	return "migrates a market data file to an adjusted market data file"
}
func (*marketCmd) Usage() string {
	return `migrate market -in <source_market_file> -out <destination_market_file>

Creates a new market data file with split-adjusted prices. The input and output files must be in different directories to prevent accidental data loss.
`
}
func (c *marketCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.in, "in", "", "The path to the source market.jsonl file which contains raw prices and split history.")
	f.StringVar(&c.out, "out", "", "The path where the new, adjusted market.jsonl will be written.")
}

func (c *marketCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.in == "" || c.out == "" {
		fmt.Fprintln(os.Stderr, "Error: -in and -out flags are required.")
		return subcommands.ExitUsageError
	}
	if filepath.Dir(c.in) == filepath.Dir(c.out) {
		fmt.Fprintln(os.Stderr, "Error: -in and -out files must not be in the same directory.")
		return subcommands.ExitUsageError
	}

	sourceMarket, err := portfolio.DecodeMarketData(c.in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding source market data: %v\n", err)
		return subcommands.ExitFailure
	}

	adjustedMarket := portfolio.NewMarketData()
	fmt.Println("Fetching adjusted prices from EODHD...")

	for sec := range sourceMarket.Securities() {
		fmt.Printf("  Fetching adjusted data for %s (%s)...\n", sec.Ticker(), sec.ID())
		adjustedMarket.Add(sec)

		//copy values from sourceMarket
		for day, price := range sourceMarket.Prices(sec.ID()) {
			adjustedMarket.Append(sec.ID(), day, price)
		}
		// and proceed to update with adjusted data
	}
	// To fetch adjusted prices
	portfolio.AdjustedPrices = true
	defer func() { portfolio.AdjustedPrices = false }()
	if err := adjustedMarket.UpdatePrices(date.New(2000, 1, 1), date.Today()); err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching adjusted prices: %v\n", err)
		return subcommands.ExitFailure
	}

	// This is the crucial part: the new market data has no splits.
	if err := portfolio.EncodeMarketData(c.out, adjustedMarket); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding adjusted market data: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("\nSuccessfully created adjusted market file at %s\n", c.out)
	return subcommands.ExitSuccess
}

// --- ledgerCmd ---

type ledgerCmd struct {
	in     string
	out    string
	market string
}

func (*ledgerCmd) Name() string     { return "ledger" }
func (*ledgerCmd) Synopsis() string { return "migrates a ledger from adjusted to raw" }
func (*ledgerCmd) Usage() string {
	return `migrate ledger -in <source_adjusted_ledger> -out <destination_raw_ledger> -market <raw_market_file>

Converts an adjusted ledger file to a raw ledger file by "un-adjusting" transactions based on split history.
`
}
func (c *ledgerCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.in, "in", "", "The path to the source, manually adjusted transactions.jsonl.")
	f.StringVar(&c.out, "out", "", "The path where the new, raw transactions.jsonl will be written.")
	f.StringVar(&c.market, "market", "", "The path to the raw market.jsonl file that contains the complete split history.")
}

func (c *ledgerCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.in == "" || c.out == "" || c.market == "" {
		fmt.Fprintln(os.Stderr, "Error: -in, -out, and -market flags are required.")
		return subcommands.ExitUsageError
	}

	adjustedLedger, err := DecodeLedger(c.in)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding adjusted ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	rawMarket, err := portfolio.DecodeMarketData(c.market)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding raw market data: %v\n", err)
		return subcommands.ExitFailure
	}

	rawLedger := portfolio.NewLedger()
	ids := make(map[string]portfolio.ID)
	for _, tx := range adjustedLedger.Transactions() {
		var newTx portfolio.Transaction = tx

		switch v := tx.(type) {
		case portfolio.Declare:
			ids[v.Ticker] = v.ID
		case portfolio.Buy:
			newTx = unadjustBuy(v, ids[v.Security], rawMarket)
		case portfolio.Sell:
			newTx = unadjustSell(v, ids[v.Security], rawMarket)
		}
		rawLedger.Append(newTx)
	}

	file, err := os.Create(c.out)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output file: %v\n", err)
		return subcommands.ExitFailure
	}
	defer file.Close()

	if err := portfolio.EncodeLedger(file, rawLedger); err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding raw ledger: %v\n", err)
		return subcommands.ExitFailure
	}

	fmt.Printf("Successfully migrated ledger to %s\n", c.out)
	return subcommands.ExitSuccess
}

func unadjustBuy(tx portfolio.Buy, secID portfolio.ID, market *portfolio.MarketData) portfolio.Buy {
	// This part needs the real split data from market.go
	// splits := market.Splits(secID)
	splits := market.Splits(secID)

	if len(splits) == 0 {
		return tx // No splits found, return as-is)
	}

	adjQuantity := decimal.NewFromFloat(tx.Quantity)
	adjusted := false

	for _, split := range splits {
		if split.Date.After(tx.When()) {
			num := decimal.NewFromInt(split.Numerator)
			den := decimal.NewFromInt(split.Denominator)
			// Un-adjust: reverse the split
			adjQuantity = adjQuantity.Mul(den).Div(num)
			adjusted = true

		}
	}
	if !adjusted {
		return tx
	}

	log.Printf("unadjusting %s %s %s from %v to %s", tx.Date, tx.Command, tx.Security, tx.Quantity, adjQuantity.String())
	tx.Quantity, _ = adjQuantity.Float64()
	return tx
}

func unadjustSell(tx portfolio.Sell, secID portfolio.ID, market *portfolio.MarketData) portfolio.Sell {
	splits := market.Splits(secID)

	if len(splits) == 0 {
		return tx // No splits found, return as-is)
	}

	adjQuantity := decimal.NewFromFloat(tx.Quantity)
	adjusted := false

	for _, split := range splits {
		if split.Date.After(tx.When()) {
			num := decimal.NewFromInt(split.Numerator)
			den := decimal.NewFromInt(split.Denominator)
			// Un-adjust: reverse the split
			adjQuantity = adjQuantity.Mul(den).Div(num)
			adjusted = true
		}
	}
	if !adjusted {
		return tx
	}

	log.Printf("unadjusting %s %s %s from %v to %s", tx.Date, tx.Command, tx.Security, tx.Quantity, adjQuantity.String())
	tx.Quantity, _ = adjQuantity.Float64()
	return tx
}

// --- checkCmd ---

type checkCmd struct {
	adjustedLedger string
	adjustedMarket string
	rawLedger      string
	rawMarket      string
}

func (*checkCmd) Name() string     { return "check" }
func (*checkCmd) Synopsis() string { return "verifies the migration by comparing reports" }
func (*checkCmd) Usage() string {
	return `migrate check -adjusted-ledger <path> -adjusted-market <path> -raw-ledger <path> -raw-market <path>

Compares the portfolio state between the adjusted and raw data sets on critical dates (around stock splits) to verify the migration.
`
}
func (c *checkCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.adjustedLedger, "adjusted-ledger", "", "Path to the original, adjusted ledger file.")
	f.StringVar(&c.adjustedMarket, "adjusted-market", "", "Path to the generated adjusted market file.")
	f.StringVar(&c.rawLedger, "raw-ledger", "", "Path to the generated raw ledger file.")
	f.StringVar(&c.rawMarket, "raw-market", "", "Path to the original market file with raw prices and splits.")
}

func (c *checkCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if c.adjustedLedger == "" || c.adjustedMarket == "" || c.rawLedger == "" || c.rawMarket == "" {
		fmt.Fprintln(os.Stderr, "Error: all four file path flags are required.")
		return subcommands.ExitUsageError
	}

	// Load all data
	adjLedger, errA := DecodeLedger(c.adjustedLedger)
	adjMarket, errB := portfolio.DecodeMarketData(c.adjustedMarket)
	rawLedger, errC := DecodeLedger(c.rawLedger)
	rawMarket, errD := portfolio.DecodeMarketData(c.rawMarket)
	if err := errors.Join(errA, errB, errC, errD); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading data files: %v\n", err)
		return subcommands.ExitFailure
	}

	// Create Accounting Systems
	asAdjusted, err := portfolio.NewAccountingSystem(adjLedger, adjMarket, "EUR")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating adjusted accounting system: %v\n", err)
		return subcommands.ExitFailure
	}
	asRaw, err := portfolio.NewAccountingSystem(rawLedger, rawMarket, "EUR")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating raw accounting system: %v\n", err)
		return subcommands.ExitFailure
	}
	fmt.Println(" Ticker  |    Date   | Adjusted Data                    | Raw Data")
	fmt.Println("--------------------------------------------------------------------------------------")

	for sec := range rawLedger.Declared() {
		// skipping securities without splits
		splits := rawMarket.Splits(sec.ID())
		if len(splits) == 0 {
			log.Println("ignoring security without splits", sec.Ticker())
			continue
		}
		ticker := sec.Ticker()
		log.Println("checking on", sec.Ticker())

		for _, tx := range rawLedger.Transactions(portfolio.BySecurity(ticker)) {
			day := tx.When()

			portfolio.AdjustedPrices = true
			reportAdj, err1 := asAdjusted.NewHoldingReport(day)
			portfolio.AdjustedPrices = false
			reportRaw, err2 := asRaw.NewHoldingReport(day)
			if err1 != nil || err2 != nil {
				fmt.Printf("Error generating reports: AdjErr: %v, RawErr: %v\n", err1, err2)
				continue
			}

			// Render reports for comparison (simplified)
			var adjOutput, rawOutput string
			var adjValue, rawValue float64
			for _, h := range reportAdj.Securities {
				if h.Ticker == ticker {
					adjOutput = fmt.Sprintf("Qty: %.4f, Val: %.2f", h.Quantity, h.MarketValue)
					adjValue = h.MarketValue
				}
			}
			for _, h := range reportRaw.Securities {
				if h.Ticker == ticker {
					rawOutput = fmt.Sprintf("Qty: %.4f, Val: %.2f", h.Quantity, h.MarketValue)
					rawValue = h.MarketValue
				}
			}
			ok := almostEqual(adjValue, rawValue, 0.1)
			if !ok {
				fmt.Printf("%-8s| %-10s| %-30s | %-30s\n", ticker, day.String(), adjOutput, rawOutput)
				break
			}
		}

	}

	return subcommands.ExitSuccess
}

// --- Helper Functions ---

// DecodeLedger decodes the ledger from a specific file path.
func DecodeLedger(path string) (*portfolio.Ledger, error) {
	file, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return portfolio.NewLedger(), nil
		}
		return nil, fmt.Errorf("could not open ledger file %q: %w", path, err)
	}
	defer file.Close()
	return portfolio.DecodeLedger(file)
}

// --- Sorting helper for splits ---
func sortSplits(splits []portfolio.Split) {
	sort.Slice(splits, func(i, j int) bool {
		return splits[i].Date.Before(splits[j].Date)
	})
}

// almostEqual compares two floats for approximate equality using a relative tolerance.
func almostEqual(a, b, tolerance float64) bool {
	if a == b {
		return true
	}
	// Avoid division by zero if the expected value is zero
	if a == 0 {
		return math.Abs(b) < tolerance
	}
	diff := math.Abs(a - b)
	return (diff / math.Abs(a)) < tolerance
}
