package eodhd

import (
	"fmt"

	"github.com/etnz/portfolio"
)

// nice to redirect to https://eodhd.com/financial-summary/00XN.XETRA

func findPrices(apiKey string, id portfolio.ID, ticker string, from, to portfolio.Date, prices map[point]PriceChange) (err error) {
	if id.IsCurrencyPair() {
		open := make(map[point]PriceChange)
		err = fetchPrices(apiKey, id, ticker, from.Add(1), to.Add(1), open, nil)
		if err != nil {
			return err
		}
		// eodhd forex sucks, the so called close value is probably buggy and equal to the open most of the time.
		// Instead the open of the next day is the closer to the truth, so be it.
		for pt, v := range open {
			prices[point{pt.Date.Add(-1), pt.ID}] = PriceChange{
				Date: v.Date,
				ID:   v.ID,
				Old:  nil,
				New:  v.New,
			}
		}
		return nil
	}

	err = fetchPrices(apiKey, id, ticker, from, to, nil, prices)
	if err != nil {
		return err
	}

	return nil
}

func findTicker(apiKey string, sec portfolio.Security) (ticker string, err error) {
	id := sec.ID()

	// Check all public types of securities.
	if id.IsCurrencyPair() {
		fromCurrency, toCurrency, _ := id.CurrencyPair()
		// The Ticker for forex is in the format "fromCurrency+toCurrency.FOREX".
		return fmt.Sprintf("%s%s.FOREX", fromCurrency, toCurrency), nil
	}

	// Determine the eodhd "exchange" and isin

	// default exchange is for funds
	exchange := "EUFUND" // see https://eodhd.com/financial-apis/covered-tickers-eodhd
	var isin string

	if i, mic, err := id.MSSI(); err == nil {
		mic2exchange, err := fetchMicToExchangeCode(apiKey)
		if err != nil {
			return "", err
		}
		exchange = mic2exchange[mic]
		isin = i
	} else if i, err := id.ISIN(); err == nil {
		isin = i
	}

	if isin == "" {
		return "", fmt.Errorf("asset %s is not traded in eodhd's exchange %s", isin, exchange)
	}

	// now fetches all tickers from that exchange
	tickers, err := fetchTickers(apiKey, exchange, false)
	if err != nil {
		// try with delisted
		tickers, err = fetchTickers(apiKey, exchange, true)
		if err != nil {
			return "", err
		}
	}
	// Search the ticker by isin now.
	for _, t := range tickers {
		if t.Isin == isin {
			// Caveat; t contains the physical exchange (e.g NASDAQ)
			// but we want the virtual exchange that speaks to eodhd
			// and that is 'exchange'. weird but that's the truth.
			return t.Code + "." + exchange, nil
		}
	}
	return "", fmt.Errorf("asset %s is not traded in eodhd's exchange %s", isin, exchange)
}
