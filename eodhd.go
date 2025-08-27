package portfolio

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/etnz/portfolio/date"
)

const eodhd_api_key = "EODHD_API_KEY"

var eodhdApiFlag = flag.String("eodhd-api-key", "", "EODHD API key to use for fetching prices from EODHD.com. This flag takes precedence over the "+eodhd_api_key+" environment variable. You can get one at https://eodhd.com/")

// eodhdApiKey retrieves the EODHD API key from the command-line flag or the environment variable.
// It prioritizes the flag over the environment variable.
func eodhdApiKey() string {
	// If the flag is not set, we try to read it from the environment variable.
	if *eodhdApiFlag == "" {
		*eodhdApiFlag = os.Getenv(eodhd_api_key)
	}
	return *eodhdApiFlag
}

// diskCache implements a simple disk cache for HTTP responses
type diskCache struct {
	base http.RoundTripper
}

// RoundTrip implements the http.RoundTripper interface. It checks for a cached
// response on disk first. If a fresh cached response is not found, it proceeds
// with the actual HTTP request and caches the new response if it's successful.
func (c *diskCache) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	// get from disk
	// diskcache implements a unique key per day, so the local tmp expires every day.
	key := fmt.Sprintf("%s %s %s", date.Today().String(), req.Method, req.URL.String())
	key = fmt.Sprintf("%x", sha1.Sum([]byte(key)))
	//key = url.PathEscape(key)

	cachedResp, err := c.get(key, req)
	if err == nil { // Cache hit
		return cachedResp, nil
	}

	resp, err = c.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	log.Printf("%v %v/%v %v", resp.Request.Method, resp.Request.URL.Host, resp.Request.URL.Path, resp.Status)
	if resp.StatusCode >= 300 {
		return resp, nil
	}
	// otherwise attempt to store it in cache

	err = c.put(key, resp)
	if err != nil {
		log.Printf("cache write err (ignored): %v\n", err)
	}
	return resp, nil
}

// get retrieves a cached response from disk
func (c *diskCache) get(key string, req *http.Request) (resp *http.Response, err error) {
	file := filepath.Join(os.TempDir(), key)
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	return http.ReadResponse(bufio.NewReader(bytes.NewBuffer(content)), req)
}

// put stores a response to disk cache
func (c *diskCache) put(key string, resp *http.Response) (err error) {
	file := filepath.Join(os.TempDir(), key)

	content, err := httputil.DumpResponse(resp, true)
	if err != nil {
		return err
	}

	f, err := os.Create(file)
	if err != nil {
		return err
	}

	_, err = f.Write(content)
	f.Close()
	return err
}

// newDailyCachingClient returns an http.Client that uses a disk cache where entries expire daily.
func newDailyCachingClient() *http.Client {
	client := new(http.Client)
	client.Transport = &diskCache{http.DefaultTransport}
	return client
}

// jwget performs an HTTP GET request and unmarshals the JSON response into the provided data structure.
func jwget(client *http.Client, addr string, data interface{}) error {
	resp, err := client.Get(addr)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("cannot http GET %v/%v: %v", resp.Request.URL.Host, resp.Request.URL.Path, resp.Status)
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return json.Unmarshal(buf.Bytes(), data)
}

// This file contains functions to access the EODHD API.

// eodhdMicToExchangeCode returns a map of MIC to EODHD's internal exchange code.
func eodhdMicToExchangeCode(apiKey string) (map[string]string, error) {
	// https://eodhd.com/api/exchanges-list/?api_token=67adc13417e148.00145034&fmt=json
	// we can retrive the Code by MIC success (add MIC to the security information (isin+mic))
	// [
	// {
	// 	"Name": "Frankfurt Exchange",
	// 	"Code": "F",
	// 	"OperatingMIC": "XFRA",
	// 	"Country": "Germany",
	// 	"Currency": "EUR",
	// 	"CountryISO2": "DE",
	// 	"CountryISO3": "DEU"
	//   },

	addr := "https://eodhd.com/api/exchanges-list/?fmt=json&api_token=" + apiKey

	// the response is a list of exchanges, each with a Code and OperatingMIC
	type Info struct {
		Code         string
		OperatingMIC string // could be a comma separated list of MICs
	}

	// that's the paylod
	content := make([]Info, 0)
	// query that endpoint at most once a day
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		return nil, err
	}
	result := make(map[string]string)
	for _, info := range content {
		for _, mic := range strings.Split(info.OperatingMIC, ",") {
			result[strings.TrimSpace(mic)] = info.Code
		}
	}
	return result, nil
}

// eodhdSearchByMSSI searches by ISIN and MIC to get the internal EODHD ticker (e.g., "AAPL.US").
func eodhdSearchByMSSI(apiKey string, isin, mic string) (ticker string, err error) {
	// https://eodhd.com/api/search/US67066G1040?api_token=67adc13417e148.00145034&fmt=json
	// [
	//   {
	//     "Code": "NVDA",
	//     "Exchange": "US",
	//     "Name": "NVIDIA Corporation",
	//     "Type": "Common Stock",
	//     "Country": "USA",
	//     "Currency": "USD",
	//     "ISIN": "US67066G1040",
	//     "previousClose": 131.14,
	//     "previousCloseDate": "2025-02-12"
	//   },

	// delisted exception for now.
	// TODO: find the way to get delisted stocks in EODHD.
	if isin == "US90184L1026" {
		return "TWTR.US", nil
	}

	mic2exchange, err := eodhdMicToExchangeCode(apiKey)
	if err != nil {
		return "", err
	}
	exchange2mic := make(map[string][]string)
	for k, v := range mic2exchange {
		exchange2mic[v] = append(exchange2mic[v], k)
	}

	exchange, ok := mic2exchange[mic]
	if !ok {
		return "", fmt.Errorf("unsupported mic %q in eodhd,com", mic)
	}

	addr := fmt.Sprintf("https://eodhd.com/api/search/%s?fmt=json&api_token=%s", url.PathEscape(isin), apiKey)
	type Info struct {
		Code          string
		Exchange      string
		Name          string
		Currency      string
		PreviousClose float64 `json:"previousClose"`
	}

	// that's the payload
	content := make([]Info, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		return "", err
	}
	for _, info := range content {
		if info.Exchange == exchange {
			// that the right ticker
			return info.Code + "." + exchange, nil
		}
	}
	for _, info := range content {
		log.Printf("Security %#v -> mics=%v", info, exchange2mic[info.Exchange])
	}
	return "", fmt.Errorf("security %s.%s is not available in eodhd.com (%d securities matching that isin)", isin, mic, len(content))
}

// eodhdSearchByISIN searches fund by ISIN to get the internal EODHD ticker (e.g., "AAPL.US").
func eodhdSearchByISIN(apiKey string, isin string) (ticker string, err error) {
	// https://eodhd.com/api/search/US67066G1040?api_token=67adc13417e148.00145034&fmt=json
	// [
	//   {
	//     "Code": "NVDA",
	//     "Exchange": "US",
	//     "Name": "NVIDIA Corporation",
	//     "Type": "Common Stock",
	//     "Country": "USA",
	//     "Currency": "USD",
	//     "ISIN": "US67066G1040",
	//     "previousClose": 131.14,
	//     "previousCloseDate": "2025-02-12"
	//   },

	addr := fmt.Sprintf("https://eodhd.com/api/search/%s?fmt=json&api_token=%s", url.PathEscape(isin), apiKey)
	type Info struct {
		Code          string
		Exchange      string
		Name          string
		Currency      string
		PreviousClose float64 `json:"previousClose"`
	}

	// that's the payload
	content := make([]Info, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		return "", err
	}
	if len(content) == 0 {
		return "", fmt.Errorf("security %s is not available in eodhd.com", isin)
	}
	if len(content) > 1 {
		return "", fmt.Errorf("security %s is not unique in eodhd.com", isin)
	}

	return content[0].Code + "." + content[0].Exchange, nil
}

// eodhdDailyMSSI returns the daily prices for a given ISIN and MIC.
func eodhdDailyMSSI(apiKey, isin, mic string, from, to date.Date) (prices date.History[float64], err error) {
	// find the eodhd ticker for the given isin and mic
	ticker, err := eodhdSearchByMSSI(apiKey, isin, mic)
	if err != nil {
		return prices, fmt.Errorf("eodhd cannot find a ticker for %s.%s: %w", isin, mic, err)
	}
	_, prices, err = eodhdDaily(apiKey, ticker, from, to)
	return prices, err
}

// eodhdDailyCurrencyPair returns the daily prices for a given currency pair.
func eodhdDailyCurrencyPair(apiKey, fromCurrency, toCurrency string, from, to date.Date) (prices date.History[float64], err error) {
	// The Ticker for forex is in the format "fromCurrency+toCurrency.FOREX".
	ticker := fmt.Sprintf("%s%s.FOREX", fromCurrency, toCurrency)
	open, _, err := eodhdDaily(apiKey, ticker, from, to)
	if err != nil {
		return prices, err
	}
	// eodhd forex sucks, the so called close value is probably buggy and equal to the open most of the time.
	// Instead the open of the next day is the closer to the truth, so be it.
	var close date.History[float64]
	for t, v := range open.Values() {
		close.Append(t.Add(-1), v)
	}
	return close, nil
}

// eodhdDailyISIN returns the daily prices for a given ISIN and MIC.
func eodhdDailyISIN(apiKey, isin string, from, to date.Date) (prices date.History[float64], err error) {
	// find the eodhd ticker for the given isin and mic
	ticker, err := eodhdSearchByISIN(apiKey, isin)
	if err != nil {
		return prices, fmt.Errorf("eodhd cannot find a ticker for %s: %w", isin, err)
	}
	_, prices, err = eodhdDaily(apiKey, ticker, from, to)
	return prices, err
}

// eodhdDaily returns the daily open and adjusted close prices for a given EODHD ticker.
// The EODHD ticker format is typically "SYMBOL.EXCHANGECODE".
func eodhdDaily(apiKey, ticker string, from, to date.Date) (open, close date.History[float64], err error) {
	// https://eodhd.com/api/eod/NVD.F?api_token=67adc13417e148.00145034&fmt=json
	// [
	//
	//	{
	//		"date": "2024-02-13",
	//		"open": 675.066,
	//		"high": 684.219,
	//		"low": 648.659,
	//		"close": 668.445,
	//		"adjusted_close": 67.705,
	//		"volume": 0
	//	  },

	// nota bene: the api also supports from and to – the format is ‘YYYY-MM-DD’.
	// If you need data from Jan 5, 2017, to Feb 10, 2017, you should use from=2017-01-05 and to=2017-02-10.
	// This should come handy to get the full range of prices.
	// However right now we don't know the what that range should be.
	// bounds are included in the response, and time is limited to 1 year with free subscription.

	addr := fmt.Sprintf("https://eodhd.com/api/eod/%s?fmt=json&api_token=%s&from=%s&to=%s", ticker, apiKey, from, to)
	type Info struct {
		Date  date.Date `json:"date"`
		Close float64   `json:"close"`
		Open  float64   `json:"open"`
	}

	// that's the payload
	content := make([]Info, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		return open, close, err
	}

	for _, info := range content {
		close.Append(info.Date, info.Close)
		open.Append(info.Date, info.Open)
	}
	return
}

// SearchResult matches the structure of a single item in the EODHD search API response.
type SearchResult struct {
	Code              string    `json:"Code"`
	Exchange          string    `json:"Exchange"`
	Name              string    `json:"Name"`
	Type              string    `json:"Type"`
	Country           string    `json:"Country"`
	Currency          string    `json:"Currency"`
	ISIN              string    `json:"ISIN"`
	PreviousClose     float64   `json:"previousClose"`
	PreviousCloseDate date.Date `json:"previousCloseDate"`
	MIC               string    `json:"-"` // Populated by Search, not from API directly.
}

// Search searches for securities via EOD Historical Data API.
func Search(searchTerm string) ([]SearchResult, error) {
	apiKey := eodhdApiKey()
	apiURL := fmt.Sprintf("https://eodhistoricaldata.com/api/search/%s?api_token=%s&fmt=json", url.PathEscape(searchTerm), url.QueryEscape(apiKey))

	var results []SearchResult
	if err := jwget(newDailyCachingClient(), apiURL, &results); err != nil {
		return nil, err
	}
	// Search results reference an exchange code that could match multiple MIC (only for the US apparently).
	mic2Exchange, err := eodhdMicToExchangeCode(apiKey)
	if err != nil {
		return nil, err
	}
	// Reverse the map.
	exchange2mic := make(map[string][]string)
	for k, v := range mic2Exchange {
		exchange2mic[v] = append(exchange2mic[v], k)
	}

	// Now we fully rebuild the search result list with potentially different MIC
	newResults := make([]SearchResult, 0, len(results))
	for _, result := range results {
		for _, mic := range exchange2mic[result.Exchange] {
			r := result
			r.MIC = mic
			newResults = append(newResults, r)
		}
	}
	return newResults, nil
}
