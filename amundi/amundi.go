package amundi

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/etnz/portfolio"
)

const amundiSessionFile = "pcs-amundi-session"

var (
	uriDispositifs  = "https://epargnant.amundi-ee.com/api/individu/arbitrages/dispositifsEligibles"
	uriDispositif   = "https://epargnant.amundi-ee.com/api/individu/produitsEpargne/idDispositif/"
	uriAffiliations = "https://epargnant.amundi-ee.com/api/individu/affiliations"
	uriAffiliation  = "https://epargnant.amundi-ee.com/api/individu/produitsEpargne/affiliation/"
)

// --- Public API ---

// Fetch retrieves market data from Amundi for the requested securities and date ranges.
// implementation always filled the response with updates from all existing securities.
func Fetch(requests map[portfolio.ID]portfolio.Range) (map[portfolio.ID]portfolio.ProviderResponse, error) {
	sessionPath := filepath.Join(os.TempDir(), amundiSessionFile)
	headerData, err := os.ReadFile(sessionPath)
	if err != nil {
		return nil, fmt.Errorf("amundi session not found. Please run 'pcs amundi-login' first: %w", err)
	}

	headers := make(http.Header)
	scanner := bufio.NewScanner(strings.NewReader(string(headerData)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			headers.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
		}
	}
	responses := make(map[portfolio.ID]portfolio.ProviderResponse)

	f := fetcher{header: headers, response: responses, request: requests}
	f.Fetch()

	return responses, nil
}

// --- Internal Logic ---

type fetcher struct {
	header   http.Header
	request  map[portfolio.ID]portfolio.Range
	response map[portfolio.ID]portfolio.ProviderResponse
}

func (f fetcher) Start() portfolio.Date {
	// The way amundi fetcher works is to scan the accounts for each day, and
	// set the price for the assets held that day.
	// There is no way to get anything outside of those assets, therefore we only
	// need to scan from the latest know date forward. That is the latest From.
	var newest portfolio.Date
	for _, r := range f.request {
		if r.From.After(newest) {
			newest = r.From
		}
	}
	return newest
}

func (f fetcher) End() portfolio.Date {
	var newest portfolio.Date
	for _, r := range f.request {
		if r.To.After(newest) {
			newest = r.To
		}
	}
	return newest
}

// appendMarketPoint add the (ticker, day, price) found from amundi portal.
func (f *fetcher) appendMarketPoint(codeFonds string, day portfolio.Date, price float64) error {
	id, err := portfolio.NewPrivate("Amundi-" + codeFonds)
	if err != nil {
		return fmt.Errorf("cannot create ID from fund name %q: %w", codeFonds, err)
	}
	if f.response == nil {
		f.response = make(map[portfolio.ID]portfolio.ProviderResponse)
	}
	prices, exists := f.response[id]
	if !exists {
		prices = portfolio.ProviderResponse{
			Prices: make(map[portfolio.Date]float64),
		}
	}
	prices.Prices[day] = price
	f.response[id] = prices
	log.Println("appending market point", id, day, price)
	return nil

}

func (f *fetcher) Fetch() error { // 	query amundi to retrieve portfolios (will do it later for affiliations)
	data, err := f.query(uriDispositifs)
	if err != nil {
		return err
	}
	start, end := f.Start(), f.End()
	if start.IsZero() || end.IsZero() || start.After(end) {
		// No valid date range to fetch.
		return nil
	}

	// Using header access the amundi portal and scan the list of "dispositifs" and update the vl found there.
	disps, err := parseAmundiDispositifs(data)
	if err != nil {
		return err
	}
	for _, dispo := range disps {
		fmt.Println("updating", dispo.Name, dispo.ID)

		// fetch days starting from start to end
		for day := start; !day.After(end); day = day.Add(1) {
			// query the dispositif detailed info (if only the api was REST I could find the next uri inside it)
			uri := uriDispositif + url.PathEscape(dispo.ID) + "?date=" + url.QueryEscape(day.Format("2006-01-02T15:04:05Z"))
			data, err := f.query(uri)
			if err != nil {
				log.Println("error querying", dispo.ID, day, err)
				continue
			}
			if err := f.parseAmundiSnapshot(data); err != nil {
				log.Println("error parsing data", string(data), err)
				continue
			}
		}
	}

	// 	query amundi to retrieve affiliations (whatever that is)
	data, err = f.query(uriAffiliations)
	if err != nil {
		return err
	}

	// Using header access the amundi portal and scan the list of "dispositifs" and update the vl found there.
	affiliations, err := parseAmundiAffiliations(data)
	if err != nil {
		return err
	}
	for _, affiliation := range affiliations {
		fmt.Println("updating", affiliation.Name, affiliation.ID)

		// fetch days starting from start to end
		for day := start; day.Before(end); day = day.Add(1) {
			// query the dispositif detailed info (if only the api was REST I could find the next uri inside it)
			uri := uriAffiliation + url.PathEscape(affiliation.ID) + "?date=" + url.QueryEscape(day.Format("2006-01-02T15:04:05Z"))
			data, err := f.query(uri)
			if err != nil {
				log.Println("error querying", affiliation.ID, day, err)
				continue
			}
			if err := f.parseAmundiSnapshot(data); err != nil {
				log.Println("error parsing data", string(data), err)
				continue
			}
		}
	}
	return nil
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
func (f *fetcher) query(uri string) ([]byte, error) {
	log.Println("querying", uri)
	r, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, fmt.Errorf("cannot create http request %q: %w", uri, err)
	}
	r.Header = f.header

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
func (f *fetcher) parseAmundiSnapshot(data []byte) (err error) {
	var snapshot struct {
		Fonds []struct {
			CodeFonds    string         `json:"codeFonds"`
			LibelleFonds string         `json:"libelleFonds"`
			VL           float64        `json:"vl"`
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
		if err := f.appendMarketPoint(ticker, day, price); err != nil {
			fmt.Fprintln(os.Stderr, err)
			errs = errors.Join(errs, err)
			continue
		}
	}
	return errs
}
