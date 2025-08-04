package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/etnz/portfolio/date"
	"github.com/google/subcommands"
)

var amundiPorfolios []string = []string{
	"https://epargnant.amundi-ee.com/api/individu/produitsEpargne/idDispositif/1-O8OPK2?date=",
	"https://epargnant.amundi-ee.com/api/individu/produitsEpargne/idDispositif/1-OTGU37?date=",
	"https://epargnant.amundi-ee.com/api/individu/produitsEpargne/affiliation/A-256714?date=",
}

// HeaderVal support dynamic -H options to mimic curl command line
type HeaderVal struct {
	buf    bytes.Buffer
	Header *http.Header
}

func (h *HeaderVal) Set(val string) error {

	reader := bufio.NewReader(strings.NewReader(h.buf.String() + val + "\n\n"))
	tp := textproto.NewReader(reader)

	mimeHeader, err := tp.ReadMIMEHeader()
	if err != nil {
		return err
	}
	// no error accept the val
	fmt.Fprintln(&h.buf, val)
	*h.Header = http.Header(mimeHeader)
	return nil
}
func (h *HeaderVal) String() string { return h.buf.String() }

type amundiCmd struct {
	header http.Header
}

func (*amundiCmd) Name() string     { return "amundi" }
func (*amundiCmd) Synopsis() string { return "import transactions from an amundi jsonl file" }
func (*amundiCmd) Usage() string {
	return `amundi <file.jsonl>:
  Import transactions from an Amundi-specific JSONL file.
`
}

func (c *amundiCmd) SetFlags(f *flag.FlagSet) {
	f.Var(&HeaderVal{Header: &c.header}, "H", "pass headers to run the uri (use chrome copy as curl to help)")
	s := "" // we complitely ignore those curl parameters.
	f.StringVar(&s, "b", "", "used by curl, but ignored here")
	f.StringVar(&s, "curl", "", "used to fake a curl, but ignored here")
}

func (c *amundiCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	start := date.New(2020, 10, 1)
	end := date.Today().Add(-1)

	market, err := DecodeSecurities()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading securities: %v\n", err)
		return subcommands.ExitFailure
	}
	var errs error
	for _, tmpl := range amundiPorfolios {
		log.Println("importing from", tmpl)
		for day := start; day.Before(end); day = day.Add(1) {
			log.Println("getting data for", day)
			data, err := queryTemplate(tmpl, day, c.header)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				errs = errors.Join(errs, err)
				continue
			}
			if err := parseAmundiSnapshot(market, data); err != nil {
				fmt.Fprintln(os.Stderr, err)
				errs = errors.Join(errs, err)
				continue
			}
		}
	}

	if err := EncodeMarketData(market); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}

func queryTemplate(tmpl string, t date.Date, header http.Header) ([]byte, error) {
	// tweak the req with the new url
	uri := tmpl + url.QueryEscape(t.Format("2006-01-02T15:04:05Z"))

	log.Println("querying", uri)
	r, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create http request %q: %w", uri, err)
	}
	r.Header = header

	// resp, err := daily().Do(r) // to test the code without actually spamming the server
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, fmt.Errorf("cannot execute http request: %w", err)
	}
	body := resp.Body
	defer body.Close()

	// reading in a buffer to be able to print the json in debug mode
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, body); err != nil {
		return nil, fmt.Errorf("cannot read receiving http body: %w", err)
	}

	return buf.Bytes(), nil
}

// ParseAmundiSnapshot reads a JSON payload representing a single-day portfolio
// snapshot from Amundi and extracts the security information (ticker, date, and price)
// from the first fund listed.
func parseAmundiSnapshot(market *portfolio.MarketData, data []byte) (err error) {
	var snapshot struct {
		Fonds []struct {
			CodeFonds    string    `json:"codeFonds"`
			LibelleFonds string    `json:"libelleFonds"`
			VL           float64   `json:"vl"`
			DateVL       date.Date `json:"dateVl"`
		} `json:"fonds"`
	}

	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("could not decode amundi snapshot json: %w", err)
	}

	if len(snapshot.Fonds) == 0 {
		// maybe this should not be an error but a reason to skip, there is no price on this day.
		return fmt.Errorf("no funds found in snapshot")
	}
	var errs error
	for _, fund := range snapshot.Fonds {
		ticker := fund.CodeFonds // Using the codeFonds as a unique ticker.
		day := fund.DateVL
		price := fund.VL
		log.Println("received market point", ticker, day, price)
		if err := appendMarketPoint(market, fund.LibelleFonds, ticker, day, price); err != nil {
			fmt.Fprintln(os.Stderr, err)
			errs = errors.Join(errs, err)
			continue
		}
	}
	return errs
}

func appendMarketPoint(market *portfolio.MarketData, fundName, ticker string, day date.Date, price float64) error {
	sec := market.Get(ticker)
	if sec == nil {
		id, err := portfolio.NewPrivate(fundName)
		if err != nil {
			return fmt.Errorf("cannot create ID from fund name: %w", err)
		}
		sec = portfolio.NewSecurity(id, ticker, "EUR")
		market.Add(sec)
	}
	sec.Prices().Append(day, price)
	return nil
}
