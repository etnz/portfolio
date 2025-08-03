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

var eodhdApiFlag = flag.String("eodhd-api-key", "", "EODHD API key to use for fetching prices from EODHD.com.\n If missing it will read for the environment variable \""+eodhd_api_key+"\". You can get one at https://eodhd.com/")

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

// returns a client with a cache all with daily expire
func daily() *http.Client {
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

// eodhdMicToCode returns a map of mic to eodhd internal code for exchange
func eodhdMicToCode(apiKey string) (map[string]string, error) {
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
	if err := jwget(daily(), addr, &content); err != nil {
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

// type SearchResult struct {
// 	Code              string
// 	MIC               []string
// 	Exchange          string
// 	Name              string
// 	Currency          string
// 	Type              string
// 	Country           string
// 	ISIN              string
// 	PreviousClose     float64 `json:"previousClose"`
// 	PreviousCloseDate string  `json:"previousCloseDate"`
// }

// func (pf *Portfolio) Search(query string) (results []SearchResult, err error) {
// 	addr := fmt.Sprintf("https://eodhd.com/api/search/%s?fmt=json&api_token=%s", url.PathEscape(query), pf.eodhdAPIKey)

// 	// that's the payload
// 	content := make([]SearchResult, 0)
// 	if err := jwget(new(http.Client), addr, &content); err != nil {
// 		return nil, err
// 	}

// 	mic2exchange, err := eodhdMicToCode(pf.eodhdAPIKey)
// 	if err != nil {
// 		return content, err
// 	}
// 	exchange2mic := make(map[string][]string)
// 	for k, v := range mic2exchange {
// 		exchange2mic[v] = append(exchange2mic[v], k)
// 	}
// 	for i := range content {
// 		content[i].MIC = exchange2mic[content[i].Exchange]
// 	}
// 	return content, nil
// }

// eodhdSearch by ISIN & MIC to get the internal eodhd ticker.
func eodhdSearch(apiKey string, isin, mic string) (ticker string, err error) {
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

	mic2exchange, err := eodhdMicToCode(apiKey)
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
	if err := jwget(daily(), addr, &content); err != nil {
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

// eodhdDailyISIN returns the daily prices for a given ISIN and MIC.
func eodhdDailyISIN(apiKey, isin, mic string, from, to date.Date) (prices date.History[float64], err error) {
	// find the eodhd ticker for the given isin and mic
	ticker, err := eodhdSearch(apiKey, isin, mic)
	if err != nil {
		return prices, fmt.Errorf("eodhd cannot find a ticker for %s.%s: %w", isin, mic, err)
	}
	_, prices, err = eodhdDaily(apiKey, ticker, from, to)
	return prices, err
}

// eodhdDailyFrom returns the daily prices for a given currency pair.
func eodhdDailyFrom(apiKey, fromCurrency, toCurrency string, from, to date.Date) (prices date.History[float64], err error) {
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

// eodhdDaily returns the daily prices for a given ticker.
// returns the daily close and open prices adjusted for splits.
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
		Close float64   `json:"adjusted_close"`
		Open  float64   `json:"open"`
	}

	// that's the payload
	content := make([]Info, 0)
	if err := jwget(daily(), addr, &content); err != nil {
		return open, close, err
	}

	for _, info := range content {
		close.Append(info.Date, info.Close)
		open.Append(info.Date, info.Open)
	}
	return
}
