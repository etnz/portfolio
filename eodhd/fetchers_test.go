package eodhd

import (
	"testing"

	"github.com/etnz/portfolio"
)

const EodhdApiDemoKey = "demo"

func Test_fetchPrices(t *testing.T) {

	prices := make(map[point]priceChange)
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
	splits := make(map[point]splitChange)
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
	dividends := make(map[point]dividendChange)
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
