package security

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/etnz/portfolio/date"
)

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

// func EODHDUpdate(apiKey, isin, mic string, from, to date.Date) (prices date.History[float64], err error) {
// 	// Find the eodhd ticker for the given isin and mic.
// 	ticker, err := eodhdSearch(apiKey, isin, mic)
// 	if err != nil {
// 		return prices, err
// 	}

// 	_, prices, err = eodhdDaily(apiKey, ticker, from, to)
// 	return prices, err
// }
