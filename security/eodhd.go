package security

import (
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/etnz/portfolio/date"
)

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

// search by ISIN (cached daily or more)
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

func eodhdDailyISIN(apiKey, isin, mic string) (prices date.History[float64], err error) {
	// find the eodhd ticker for the given isin and mic
	ticker, err := eodhdSearch(apiKey, isin, mic)
	if err != nil {
		return prices, fmt.Errorf("eodhd cannot find a ticker for %s.%s: %w", isin, mic, err)
	}
	_, prices, err = eodhdDaily(apiKey, ticker)
	return prices, err
}
func eodhdDailyFrom(apiKey, from, to string) (prices date.History[float64], err error) {
	open, _, err := eodhdDaily(apiKey, from+to+".FOREX")
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

//https://eodhd.com/api/eod/USDEUR.FOREX?order=d&api_token=67adc13417e148.00145034&fmt=json

func eodhdDaily(apiKey, ticker string) (open, close date.History[float64], err error) {
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

	addr := fmt.Sprintf("https://eodhd.com/api/eod/%s?fmt=json&api_token=%s", ticker, apiKey)
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
