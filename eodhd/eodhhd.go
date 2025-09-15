package eodhd

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
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
	key := fmt.Sprintf("%s %s %s", portfolio.Today().String(), req.Method, req.URL.String())
	key = fmt.Sprintf("daily-%x", sha1.Sum([]byte(key)))
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

// jwget performs an HTTP GET request to the given address and unmarshals the
// JSON response body into the provided data structure. It uses the provided
// http.Client for the request.
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
func eodhdDailyMSSI(apiKey, isin, mic string, from, to portfolio.Date, prices map[portfolio.Date]float64) (err error) {
	// find the eodhd ticker for the given isin and mic
	ticker, err := eodhdSearchByMSSI(apiKey, isin, mic)
	if err != nil {
		return fmt.Errorf("eodhd cannot find a ticker for %s.%s: %w", isin, mic, err)
	}
	err = eodhdDaily(apiKey, ticker, from, to, nil, prices)
	return err
}

// eodhdDailyCurrencyPair returns the daily prices for a given currency pair.
func eodhdDailyCurrencyPair(apiKey, fromCurrency, toCurrency string, from, to portfolio.Date, prices map[portfolio.Date]float64) (err error) {
	// The Ticker for forex is in the format "fromCurrency+toCurrency.FOREX".
	ticker := fmt.Sprintf("%s%s.FOREX", fromCurrency, toCurrency)
	open := make(map[portfolio.Date]float64)
	err = eodhdDaily(apiKey, ticker, from, to, open, nil)
	if err != nil {
		return err
	}
	// eodhd forex sucks, the so called close value is probably buggy and equal to the open most of the time.
	// Instead the open of the next day is the closer to the truth, so be it.
	for date, value := range open {
		prices[date.Add(-1)] = value
	}
	return nil
}

// eodhdDailyISIN returns the daily prices for a given ISIN and MIC.
func eodhdDailyISIN(apiKey, isin string, from, to portfolio.Date, prices map[portfolio.Date]float64) (err error) {
	// find the eodhd ticker for the given isin and mic
	ticker, err := eodhdSearchByISIN(apiKey, isin)
	if err != nil {
		return fmt.Errorf("eodhd cannot find a ticker for %s: %w", isin, err)
	}
	err = eodhdDaily(apiKey, ticker, from, to, nil, prices)
	return err
}

// eodhdDaily returns the daily open and close prices for a given EODHD ticker.
// The EODHD ticker format is typically "SYMBOL.EXCHANGECODE".
func eodhdDaily(apiKey, ticker string, from, to portfolio.Date, open, close map[portfolio.Date]float64) (err error) {
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
		Date          portfolio.Date `json:"date"`
		Close         float64        `json:"close"`
		Open          float64        `json:"open"`
		AdjustedClose float64        `json:"adjusted_close"`
	}

	// that's the payload
	content := make([]Info, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		//log.Printf("failed to jwget %s: %v", addr, err)
		return err
	}

	for _, info := range content {
		if close != nil {
			close[info.Date] = info.Close
		}
		if open != nil {
			open[info.Date] = info.Open
		}
	}
	return
}

// simplifyDecimalRatio converts a ratio of decimals into a simplified integer fraction.
func simplifyDecimalRatio(numDecimal, denDecimal decimal.Decimal) (num, den int64) {
	// To convert the decimal ratio to a simple integer fraction,
	// we find a common multiplier to make both numerator and denominator integers.
	// We use the exponent of the decimal (number of digits after the decimal point).
	numExp := -numDecimal.Exponent()
	denExp := -denDecimal.Exponent()
	multiplier := decimal.NewFromInt(1)
	if numExp > 0 {
		multiplier = multiplier.Mul(decimal.NewFromInt(10).Pow(decimal.NewFromInt32(numExp)))
	}
	if denExp > numExp {
		multiplier = decimal.NewFromInt(10).Pow(decimal.NewFromInt32(denExp))
	}

	numInt := numDecimal.Mul(multiplier).BigInt()
	denInt := denDecimal.Mul(multiplier).BigInt()

	// Simplify the fraction by dividing by the greatest common divisor.
	commonDivisor := new(big.Int).GCD(nil, nil, numInt, denInt)

	num = new(big.Int).Div(numInt, commonDivisor).Int64()
	den = new(big.Int).Div(denInt, commonDivisor).Int64()
	return
}

// eodhdSplits returns the split history for a given EODHD ticker.
func eodhdSplits(apiKey, ticker string, splits map[portfolio.Date]portfolio.SplitInfo) error {
	addr := fmt.Sprintf("https://eodhd.com/api/splits/%s?fmt=json&api_token=%s", ticker, apiKey)

	type apiSplit struct {
		Date  portfolio.Date `json:"date"`
		Split string         `json:"split"`
	}

	content := make([]apiSplit, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		return err
	}

	for _, s := range content {
		parts := strings.Split(s.Split, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid split format from API: %q", s.Split)
		}

		numDecimal, err := decimal.NewFromString(parts[0])
		if err != nil {
			return fmt.Errorf("invalid numerator in split %q: %w", s.Split, err)
		}
		denDecimal, err := decimal.NewFromString(parts[1])
		if err != nil {
			return fmt.Errorf("invalid denominator in split %q: %w", s.Split, err)
		}

		num, den := simplifyDecimalRatio(numDecimal, denDecimal)
		splits[s.Date] = portfolio.SplitInfo{
			Numerator:   num,
			Denominator: den,
		}
	}
	return nil
}

// eodhdDividends returns the dividend history for a given EODHD ticker.
func eodhdDividends(apiKey, ticker string, dividends map[portfolio.Date]portfolio.DividendInfo) error {
	addr := fmt.Sprintf("https://eodhd.com/api/div/%s?fmt=json&api_token=%s", ticker, apiKey)

	type apiDividend struct {
		Date  portfolio.Date  `json:"date"` // ex-dividend date, see https://eodhd.com/financial-apis/api-splits-dividends
		Value decimal.Decimal `json:"value"`
	}

	content := make([]apiDividend, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		return err
	}

	for _, d := range content {
		dividends[d.Date] = portfolio.DividendInfo{
			Amount: d.Value.InexactFloat64(),
		}
	}
	return nil
}

// SearchResult matches the structure of a single item in the EODHD search API response.
type SearchResult struct {
	Code              string         `json:"Code"`
	Exchange          string         `json:"Exchange"`
	Name              string         `json:"Name"`
	Type              string         `json:"Type"`
	Country           string         `json:"Country"`
	Currency          string         `json:"Currency"`
	ISIN              string         `json:"ISIN"`
	PreviousClose     float64        `json:"previousClose"`
	PreviousCloseDate portfolio.Date `json:"previousCloseDate"`
	MIC               string         `json:"-"` // Populated by Search, not from API directly.
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

func Fetch(requests map[portfolio.ID]portfolio.Range) (map[portfolio.ID]portfolio.ProviderResponse, error) {
	f := fetcher{request: requests, response: make(map[portfolio.ID]portfolio.ProviderResponse)}
	return f.Fetch()
}

type fetcher struct {
	request  map[portfolio.ID]portfolio.Range
	response map[portfolio.ID]portfolio.ProviderResponse
	// Cache for security details to avoid redundant API calls.
	securities map[portfolio.ID]portfolio.Security
}

func (f *fetcher) Fetch() (map[portfolio.ID]portfolio.ProviderResponse, error) {
	apiKey := eodhdApiKey()
	if apiKey == "" {
		return nil, errors.New("EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable")
	}

	var errs error

	for id, reqRange := range f.request {
		resp := portfolio.ProviderResponse{
			Prices:    make(map[portfolio.Date]float64),
			Splits:    make(map[portfolio.Date]portfolio.SplitInfo),
			Dividends: make(map[portfolio.Date]portfolio.DividendInfo),
		}

		// Fetch prices
		if err := f.updateSecurityPrices(id, resp.Prices, reqRange.From, reqRange.To); err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to fetch prices for %s: %w", id, err))
		}

		// Fetch splits
		err := f.updateSecuritySplits(id, resp.Splits)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to fetch splits for %s: %w", id, err))
		}

		// Fetch dividends
		err = f.updateSecurityDividends(id, resp.Dividends)
		if err != nil {
			errs = errors.Join(errs, fmt.Errorf("failed to fetch dividends for %s: %w", id, err))
		}

		f.response[id] = resp
	}

	return f.response, errs

}

// updateSecurityPrices attempts to fetch and update prices for a single security.
func (*fetcher) updateSecurityPrices(id portfolio.ID, prices map[portfolio.Date]float64, from, to portfolio.Date) error {
	apiKey := eodhdApiKey()
	if apiKey == "" {
		return errors.New("EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable")
	}

	var err error

	// Determine security type and fetch prices accordingly.
	if isin, mic, mssiErr := id.MSSI(); mssiErr == nil {
		// This is an MSSI security.
		err = eodhdDailyMSSI(apiKey, isin, mic, from, to, prices)
		if err != nil {
			return fmt.Errorf("failed to get prices for MSSI %s: %w", id, err)
		}
	} else if base, quote, cpErr := id.CurrencyPair(); cpErr == nil {
		// This is a CurrencyPair.
		err = eodhdDailyCurrencyPair(apiKey, base, quote, from, to, prices)
		if err != nil {
			return fmt.Errorf("failed to get prices for CurrencyPair %s: %w", id, err)
		}
	} else if isin, fundErr := id.ISIN(); fundErr == nil {
		// This is a Fund.
		err = eodhdDailyISIN(apiKey, isin, from, to, prices)
		if err != nil {
			return fmt.Errorf("failed to get prices for ISIN %s: %w", id, err)
		}

	} else {
		// This is a private or unsupported security type for updates.
		return nil // Not an error, just nothing to do.
	}

	if len(prices) == 0 {
		log.Printf("no new prices found for security %q between %s and %s", id, from, to)
		return nil
	}
	return nil
}

func (*fetcher) updateSecuritySplits(sec portfolio.ID, splits map[portfolio.Date]portfolio.SplitInfo) error {
	apiKey := eodhdApiKey()
	if apiKey == "" {
		return errors.New("EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable")
	}

	var ticker string
	var err error

	// Determine security type and fetch ticker accordingly.
	if isin, mic, mssiErr := sec.MSSI(); mssiErr == nil {
		// This is an MSSI security.
		ticker, err = eodhdSearchByMSSI(apiKey, isin, mic)
		if err != nil {
			return fmt.Errorf("failed to get ticker for MSSI %s: %w", sec, err)
		}
	} else if isin, fundErr := sec.ISIN(); fundErr == nil {
		// This is a Fund.
		ticker, err = eodhdSearchByISIN(apiKey, isin)
		if err != nil {
			return fmt.Errorf("failed to get ticker for ISIN %s: %w", sec, err)
		}
	} else {
		// This is a private or unsupported security type for updates.
		return nil // Not an error, just nothing to do.
	}

	if ticker == "" {
		return nil // No ticker found, nothing to do.
	}

	return eodhdSplits(apiKey, ticker, splits)
}

func (*fetcher) updateSecurityDividends(sec portfolio.ID, dividends map[portfolio.Date]portfolio.DividendInfo) error {
	apiKey := eodhdApiKey()
	if apiKey == "" {
		return errors.New("EODHD API key is not set. Use -eodhd-api-key flag or EODHD_API_KEY environment variable")
	}

	var ticker string
	var err error

	// Determine security type and fetch ticker accordingly.
	if isin, mic, mssiErr := sec.MSSI(); mssiErr == nil {
		// This is an MSSI security.
		ticker, err = eodhdSearchByMSSI(apiKey, isin, mic)
		if err != nil {
			return fmt.Errorf("failed to get ticker for MSSI %s: %w", sec, err)
		}
	} else if isin, fundErr := sec.ISIN(); fundErr == nil {
		// This is a Fund.
		ticker, err = eodhdSearchByISIN(apiKey, isin)
		if err != nil {
			return fmt.Errorf("failed to get ticker for ISIN %s: %w", sec, err)
		}
	} else {
		// This is a private or unsupported security type for updates.
		return nil // Not an error, just nothing to do.
	}

	if ticker == "" {
		return nil // No ticker found, nothing to do.
	}

	return eodhdDividends(apiKey, ticker, dividends)
}
