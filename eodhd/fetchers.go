package eodhd

import (
	"fmt"
	"strings"

	"github.com/etnz/portfolio"
	"github.com/shopspring/decimal"
)

// This file contains functions to access the EODHD API.

// fetchMicToExchangeCode returns a map of MIC to EODHD's internal exchange code.
//
// This is required since EODHD use its own id for exchange places.
func fetchMicToExchangeCode(apiKey string) (map[string]string, error) {
	// https://eodhd.com/api/exchanges-list/?api_token=demo&fmt=json
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

// fetchPrices fills the daily open and close prices for a given EODHD ticker.
// The EODHD ticker format is typically "SYMBOL.EXCHANGECODE".
func fetchPrices(apiKey string, id portfolio.ID, ticker string, from, to portfolio.Date, open, close map[point]PriceChange) (err error) {
	// https://eodhd.com/api/eod/NVD.F?api_token=demo&fmt=json
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
		Date  portfolio.Date  `json:"date"`
		Close decimal.Decimal `json:"close"`
		Open  decimal.Decimal `json:"open"`
		// AdjustedClose decimal.Decimal        `json:"adjusted_close"`
	}

	// that's the payload
	content := make([]Info, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		//log.Printf("failed to jwget %s: %v", addr, err)
		return err
	}

	for _, info := range content {
		if close != nil {
			close[point{info.Date, id}] = PriceChange{
				Date: info.Date,
				ID:   id,
				Old:  nil,
				New:  info.Close,
			}
		}
		if open != nil {
			open[point{info.Date, id}] = PriceChange{
				Date: info.Date,
				ID:   id,
				Old:  nil,
				New:  info.Open,
			}
		}
	}
	return
}

// fetchSplits returns the split history for a given EODHD ticker.
func fetchSplits(apiKey string, id portfolio.ID, ticker string, from, to portfolio.Date, splits map[point]SplitChange) error {
	addr := fmt.Sprintf("https://eodhd.com/api/splits/%s?fmt=json&api_token=%s&from=%s&to=%s", ticker, apiKey, from, to)

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
		splits[point{s.Date, id}] = SplitChange{
			Date:        s.Date,
			ID:          id,
			Numerator:   num,
			Denominator: den,
		}
	}
	return nil
}

// fetchDividends returns the dividend history for a given EODHD ticker.
func fetchDividends(apiKey string, id portfolio.ID, ticker string, from, to portfolio.Date, dividends map[point]DividendChange) error {
	addr := fmt.Sprintf("https://eodhd.com/api/div/%s?fmt=json&api_token=%s&from=%s&to=%s", ticker, apiKey, from, to)

	type apiDividend struct {
		Date     portfolio.Date  `json:"date"` // ex-dividend date, see https://eodhd.com/financial-apis/api-splits-dividends
		Value    decimal.Decimal `json:"value"`
		Currency string          `json:"currency"`
	}

	content := make([]apiDividend, 0)
	if err := jwget(newDailyCachingClient(), addr, &content); err != nil {
		return err
	}

	for _, d := range content {
		dividends[point{d.Date, id}] = DividendChange{
			Date:     d.Date,
			ID:       id,
			Amount:   d.Value,
			Currency: d.Currency,
		}
	}
	return nil
}

// TickerInfo holds information about a specific ticker on an exchange from the EODHD API.
type TickerInfo struct {
	Code     string `json:"Code"`
	Name     string `json:"Name"`
	Country  string `json:"Country"`
	Exchange string `json:"Exchange"`
	Currency string `json:"Currency"`
	Type     string `json:"Type"`
	Isin     string `json:"Isin"`
}

// fetchTickers retrieves the list of all tickers for a given exchange code.
func fetchTickers(apiKey string, exchangeCode string, delisted bool) ([]TickerInfo, error) {
	// API Documentation: https://eodhd.com/api/exchange-symbol-list/{EXCHANGE_CODE}
	// Example response
	// [
	// {
	// "Code": "CDR",
	// "Name": "CD PROJEKT SA",
	// "Country": "Poland",
	// "Exchange": "WAR",
	// "Currency": "PLN",
	// "Type": "Common Stock",
	// "Isin": "PLOPTTC00011"
	// },
	// {
	// "Code": "PKN",
	// "Name": "PKN Orlen SA",
	// "Country": "Poland",
	// "Exchange": "WAR",
	// "Currency": "PLN",
	// "Type": "Common Stock",
	// "Isin": "PLPKN0000018"
	// }
	// ... ]

	addr := fmt.Sprintf("https://eodhd.com/api/exchange-symbol-list/%s?api_token=%s&fmt=json", exchangeCode, apiKey)
	if delisted {
		addr = fmt.Sprintf("https://eodhd.com/api/exchange-symbol-list/%s?api_token=%s&fmt=json&delisted=1", exchangeCode, apiKey)
	}

	var content []TickerInfo
	if err := jwget(newMonthlyCachingClient(), addr, &content); err != nil {
		return nil, fmt.Errorf("failed to fetch tickers for exchange %s: %w", exchangeCode, err)
	}

	return content, nil
}
