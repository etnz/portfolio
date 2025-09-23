package eodhd

import (
	"testing"

	"github.com/etnz/portfolio"
)

const EodhdApiDemoKey = "demo"

func Test_fetchPrices(t *testing.T) {

	prices := make(map[point]PriceChange)
	err := fetchPrices(EodhdApiDemoKey, portfolio.ID("None"), "MCD.US", portfolio.Today().Add(-10), portfolio.Today().Add(-1), nil, prices)
	if err != nil {
		t.Errorf("eodhdDailyFrom() unexpected error = %v", err)
	}
	if len(prices) == 0 {
		t.Error("eodhdDailyFrom() no prices returned")
	}
}

func Test_fetchMicToExchangeCode(t *testing.T) {
	if EodhdApiDemoKey == "demo" {
		t.Skip("not supported with demo key, use a real one.")
	}
	mic2exchange, err := fetchMicToExchangeCode(EodhdApiDemoKey)
	if err != nil {
		t.Fatalf("fetchMicToExchangeCode() unexpected error = %v", err)
	}
	if len(mic2exchange) == 0 {
		t.Error("fetchMicToExchangeCode() returned an empty map")
	}
	// Frankfurt Stock Exchange
	if code, ok := mic2exchange["XFRA"]; !ok || code != "F" {
		t.Errorf("fetchMicToExchangeCode() expected 'XFRA' to be 'F', got '%s'", code)
	}
}

func Test_fetchSplits(t *testing.T) {
	splits := make(map[point]SplitChange)
	// Using AAPL.US as it has a known split history.
	// The from/to dates are currently ignored by the function, but we pass them for future-proofing.
	err := fetchSplits(EodhdApiDemoKey, portfolio.ID("None"), "AAPL.US", portfolio.NewDate(2000, 1, 1), portfolio.Today(), splits)
	if err != nil {
		t.Errorf("fetchSplits() unexpected error = %v", err)
	}
	if len(splits) == 0 {
		t.Error("fetchSplits() no splits returned for AAPL.US, which is unexpected")
	}
}

func Test_fetchDividends(t *testing.T) {
	dividends := make(map[point]DividendChange)
	// Using AAPL.US as it has a known dividend history.
	// The from/to dates are currently ignored by the function.
	err := fetchDividends(EodhdApiDemoKey, portfolio.ID("None"), "AAPL.US", portfolio.NewDate(2023, 1, 1), portfolio.Today(), dividends)
	if err != nil {
		t.Errorf("fetchDividends() unexpected error = %v", err)
	}
	if len(dividends) == 0 {
		t.Error("fetchDividends() no dividends returned for AAPL.US, which is unexpected")
	}
}

func Test_fetchTickers(t *testing.T) {
	if EodhdApiDemoKey == "demo" {
		t.Skip("not supported with demo key, use a real one.")
	}
	// Using "F" for Frankfurt Exchange
	tickers, err := fetchTickers(EodhdApiDemoKey, "NYSE")
	if err != nil {
		t.Fatalf("fetchTickers() unexpected error = %v", err)
	}
	if len(tickers) == 0 {
		t.Error("fetchTickers() returned no tickers for exchange 'NYSE'")
	}

	// Check for a known ticker on that exchange
	found := false
	for _, ticker := range tickers {
		if ticker.Code == "TWTR" { // Adidas AG
			found = true
			break
		}
	}
	if !found {
		t.Error("fetchTickers() did not find expected ticker 'TWTR' in exchange 'NYSE'")
	}
}

func Test_Search(t *testing.T) {
	if EodhdApiDemoKey == "demo" {
		t.Skip("not supported with demo key, use a real one.")
	}
	results, err := Search(EodhdApiDemoKey, "Apple")
	if err != nil {
		t.Fatalf("Search() unexpected error = %v", err)
	}
	if len(results) == 0 {
		t.Error("Search() returned no results for 'Apple'")
	}

	found := false
	for _, res := range results {
		if res.Code == "AAPL" && res.Exchange == "US" {
			found = true
			if res.MIC == "" {
				t.Error("Search() result for AAPL.US has an empty MIC")
			}
			break
		}
	}
	if !found {
		t.Error("Search() did not find 'AAPL.US' in results for 'Apple'")
	}
}
