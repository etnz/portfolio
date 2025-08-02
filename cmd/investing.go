package cmd

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

// importInvestingCmd imports public security prices from investing's csv format.
type importInvestingCmd struct {
	file string
}

func (*importInvestingCmd) Name() string { return "import-investing" }
func (*importInvestingCmd) Synopsis() string {
	return "import public security prices in investing.com's csv format"
}
func (*importInvestingCmd) Usage() string { return "pcs import-investing <ticker>\n" }
func (c *importInvestingCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.file, "i", "", "input file")
}
func (c *importInvestingCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {
	if f.NArg() != 1 {
		fmt.Println("a security ticker is required as argument")
		return subcommands.ExitUsageError
	}
	ticker := f.Arg(0)

	if c.file == "" {
		fmt.Println("-i argument is required")
		return subcommands.ExitUsageError
	}

	db, err := OpenSecurities()
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	fmt.Printf("importing %q from %q\n", ticker, c.file)
	w, err := os.Open(c.file)
	if err != nil {
		fmt.Printf("cannot open file %q: %v\n", c.file, err)
		return subcommands.ExitFailure
	}
	defer w.Close()

	newPrices, err := parseCSV(w)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	sec := db.Get(ticker)
	if sec == nil {
		fmt.Println("unknown ticker", ticker)
		return subcommands.ExitFailure
	}
	oldPrices := sec.Prices()

	for on, price := range newPrices.Values() {
		oldPrices.Append(on, price)
	}

	if err := CloseSecurities(db); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

// "09/30/2022","43.84","42.82","44.62","42.81","25.89M","2.57%"
var csvFmt = regexp.MustCompile(`"(\d\d/\d\d/\d\d\d\d)","(\d+.\d+)"`)

// parseCSV parses the CSV file in investing.com's format and returns a date.History of prices.
// The CSV file is expected to have the following format:
// "09/30/2022","43.84","42.82","44.62","42.81","25.89M","2.57%"
// The first line is a header line and is skipped.
// The date is in the format "MM/DD/YYYY" and the price is a float64 value.
// The function returns an error if the format is invalid or if there are any parsing errors.
func parseCSV(r io.Reader) (date.History[float64], error) {
	line := 0
	scanner := bufio.NewScanner(r)
	var prices date.History[float64]

	for scanner.Scan() {
		line++
		// Skip header line
		if line == 1 {
			continue
		}

		row := string(scanner.Bytes())
		subs := csvFmt.FindStringSubmatch(row)
		if len(subs) != 3 {
			return prices, fmt.Errorf("invalid Investing csv format line %d: got %q", line, row)
		}
		sdate := subs[1]
		t, err := time.Parse("01/02/2006", sdate)
		if err != nil {
			return prices, fmt.Errorf("invalid Investing csv format line %d: invalid date %q: %w", line, sdate, err)
		}
		on := date.New(t.Date())

		sclose := subs[2]
		value, err := strconv.ParseFloat(sclose, 64)
		if err != nil {
			return prices, fmt.Errorf("invalid Investing csv format line %d: invalid number %q", line, sclose)
		}
		prices.Append(on, value)
	}
	return prices, nil
}
