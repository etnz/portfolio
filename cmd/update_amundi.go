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
	"github.com/google/subcommands"
)

var (
	uriDispositifs  = "https://epargnant.amundi-ee.com/api/individu/arbitrages/dispositifsEligibles"
	uriDispositif   = "https://epargnant.amundi-ee.com/api/individu/produitsEpargne/idDispositif/"
	uriAffiliations = "https://epargnant.amundi-ee.com/api/individu/affiliations"
	uriAffiliation  = "https://epargnant.amundi-ee.com/api/individu/produitsEpargne/affiliation/"

	// it seems that this is listing all the products which should simplify the discovery uriDispositifs and uriAffiliations
	// uriProducts = "https://epargnant.amundi-ee.com/api/individu/produitsEpargne?codeRegroupement=ER%2CRC%2CES"
)

// HeaderVal support dynamic -H options to mimic curl command line
// HeaderVal is a custom flag.Value implementation to support dynamic -H options
// similar to curl's command line. It parses header strings and populates
// an http.Header map.
type HeaderVal struct {
	buf    bytes.Buffer
	Header *http.Header
}

// Set parses a header string (e.g., "Content-Type: application/json") and
// adds it to the http.Header map. It appends the new value to an internal
// buffer to allow multiple -H flags.
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

// String returns the concatenated string of all headers set so far.
func (h *HeaderVal) String() string { return h.buf.String() }

type updateAmundiCmd struct {
	header http.Header
	start  string // the start date
}

func (*updateAmundiCmd) Name() string     { return "update-amundi" }
func (*updateAmundiCmd) Synopsis() string { return "import transactions from an amundi jsonl file" }
func (*updateAmundiCmd) Usage() string {
	return `pcs update-amundi [-start <date>] -curl <curl command arguments, only -H is used>:
  Update security prices that are only available on Amundi portal (amundi-ee.com).
  It scans all your saving accounts (for ID) then get its daily summary that contains the securities price.

  To access Amundi portal API from the CLI we need identifications headers. Here is how to proceed: 
  
  * Open your portal https://epargnant.amundi-ee.com/#/epargne?onglet=ES
  * Open the developer/Inspect the page
  * Find 1 xhr request with identification headers
  * Copy it as a curl command line (you can try it as a standalone command too)
  * Finally paste it in the command line: 'pcs update-amundi -<paste the curl command here>'
`
}

func (c *updateAmundiCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.start, "start", portfolio.Today().Add(-7).String(), "Start date. See the user manual for supported date formats.")

	f.Var(&HeaderVal{Header: &c.header}, "H", "pass headers to run the uri (use chrome copy as curl to help)")
	s := "" // we complitely ignore those curl parameters.
	f.StringVar(&s, "b", "", "used by curl, but ignored here")
	f.StringVar(&s, "curl", "", "used to fake a curl, but ignored here")
}

func (c *updateAmundiCmd) Execute(_ context.Context, f *flag.FlagSet, _ ...interface{}) subcommands.ExitStatus {

	start, err := portfolio.ParseDate(c.start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing date: %v\n", err)
		return subcommands.ExitUsageError
	}
	end := portfolio.Today()

	market, err := DecodeMarketData()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading accounting system: %v\n", err)
		return subcommands.ExitFailure
	}

	// 	query amundi to retrieve portfolios (will do it later for affiliations)
	data, err := c.query(uriDispositifs)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	// Using header access the amundi portal and scan the list of "dispositifs" and update the vl found there.
	disps, err := parseAmundiDispositifs(data)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	for _, dispo := range disps {
		fmt.Println("updating", dispo.Name, dispo.ID)

		// fetch days starting from start to end
		for day := start; !day.After(end); day = day.Add(1) {
			// query the dispositif detailed info (if only the api was REST I could find the next uri inside it)
			uri := uriDispositif + url.PathEscape(dispo.ID) + "?date=" + url.QueryEscape(day.Format("2006-01-02T15:04:05Z"))
			data, err := c.query(uri)
			if err != nil {
				log.Println("error querying", dispo.ID, day, err)
				continue
			}
			if err := parseAmundiSnapshot(market, data); err != nil {
				log.Println("error parsing data", string(data), err)
				continue
			}
		}
	}

	// 	query amundi to retrieve affiliations (whatever that is)
	data, err = c.query(uriAffiliations)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	// Using header access the amundi portal and scan the list of "dispositifs" and update the vl found there.
	affiliations, err := parseAmundiAffiliations(data)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	for _, affiliation := range affiliations {
		fmt.Println("updating", affiliation.Name, affiliation.ID)

		// fetch days starting from start to end
		for day := start; day.Before(end); day = day.Add(1) {
			// query the dispositif detailed info (if only the api was REST I could find the next uri inside it)
			uri := uriAffiliation + url.PathEscape(affiliation.ID) + "?date=" + url.QueryEscape(day.Format("2006-01-02T15:04:05Z"))
			data, err := c.query(uri)
			if err != nil {
				log.Println("error querying", affiliation.ID, day, err)
				continue
			}
			if err := parseAmundiSnapshot(market, data); err != nil {
				log.Println("error parsing data", string(data), err)
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

// dispositifInfo holds the extracted ID and name for a non-piloted dispositif.
type dispositifInfo struct {
	ID   string
	Name string
}

// parseAmundiDispositifs reads a JSON payload from an Amundi dispositifs
// export and extracts all dispositifs that are not pilot-managed ("pilote": false).
func parseAmundiDispositifs(data []byte) ([]dispositifInfo, error) {
	var payload struct {
		Dispositifs []struct {
			ID          string `json:"idDispositif"`
			Name        string `json:"libelleDispositifMetier"`
			NaturePoche string `json:"naturePoche"`
		} `json:"dispositifs"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("could not decode amundi dispositifs json: %w", err)
	}

	var results []dispositifInfo

	for _, disp := range payload.Dispositifs {
		log.Println("dispositif", disp.ID, disp.Name, disp.NaturePoche)
		if disp.NaturePoche == "" || disp.NaturePoche == "ES" {
			results = append(results, dispositifInfo{
				ID:   disp.ID,
				Name: disp.Name,
			})
		}
	}

	return results, nil
}

// affiliationInfo holds the extracted ID and contract type.
type affiliationInfo struct {
	ID   string
	Name string
}

// parseAmundiAffiliations reads a JSON payload from an Amundi affiliations
// export and extracts the ID and contract type for each affiliation.
func parseAmundiAffiliations(data []byte) ([]affiliationInfo, error) {
	var payload struct {
		Affiliations []struct {
			ID           string `json:"idAffiliation"`
			ContractType string `json:"typeContrat"`
		} `json:"affiliations"`
	}

	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("could not decode amundi affiliations json: %w", err)
	}

	var results []affiliationInfo
	for _, aff := range payload.Affiliations {
		results = append(results, affiliationInfo{
			ID:   aff.ID,
			Name: aff.ContractType,
		})
	}

	return results, nil
}

// query amundi portal with header.
func (c *updateAmundiCmd) query(uri string) ([]byte, error) {
	log.Println("querying", uri)
	r, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create http request %q: %w", uri, err)
	}
	r.Header = c.header

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
// snapshot from Amundi (same for "dispositif" and "affiliation") and extracts the security information (ticker, date, and price)
// from the first fund listed.
func parseAmundiSnapshot(market *portfolio.MarketData, data []byte) (err error) {
	var snapshot struct {
		Fonds []struct {
			CodeFonds    string    `json:"codeFonds"`
			LibelleFonds string    `json:"libelleFonds"`
			VL           float64   `json:"vl"`
			DateVL       portfolio.Date `json:"dateVl"`
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
		log.Println("received market point", ticker, day, price, fund.LibelleFonds)
		if err := appendMarketPoint(market, ticker, day, price); err != nil {
			fmt.Fprintln(os.Stderr, err)
			errs = errors.Join(errs, err)
			continue
		}
	}
	return errs
}

// appendMarketPoint add the (ticker, day, price) found from amundi portal.
func appendMarketPoint(market *portfolio.MarketData, ticker string, day portfolio.Date, price float64) error {
	id, err := portfolio.NewPrivate("Amundi-" + ticker)
	if err != nil {
		return fmt.Errorf("cannot create ID from fund name %q: %w", ticker, err)
	}

	sec := market.Get(id)
	if sec == (portfolio.Security{}) {
		sec = portfolio.NewSecurity(id, ticker, "EUR")
		market.Add(sec)
	}
	log.Println("appending market point", day, ticker, price)
	market.Append(id, day, price)
	return nil

}
