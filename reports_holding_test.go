package portfolio

import (
	"sort"
	"testing"
	"time"
)

func TestAccountingSystem_NewHoldingReport(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "EUR"
	o := NewDate(2025, time.January, 1)

	ledger.Append(
		NewDeclare(o, "", "AAPL", AAPL, "USD"),
		NewDeclare(o, "", "GOOG", GOOG, "USD"),
		NewBuy(NewDate(2025, time.January, 10), "", "AAPL", Q(100), USD(100*150.0)),
		NewBuy(NewDate(2025, time.January, 15), "", "GOOG", Q(50), USD(50*2800.0)),
		NewSell(NewDate(2025, time.February, 1), "", "AAPL", Q(25), USD(25*160.0)),
		NewDeposit(NewDate(2025, time.February, 5), "", EUR(10000), ""),
		NewUpdatePrice(NewDate(2025, time.February, 1), "AAPL", USD(160.0)),
		NewUpdatePrice(NewDate(2025, time.February, 1), "GOOG", USD(2900.0)),
		// USDEUR is not a security, it's a forex rate. The journal will create it.
		// For the test, we can add an update-price for a fake ticker.
		NewDeclare(o, "", "USDEUR", USDEUR, "EUR"),
		NewUpdatePrice(NewDate(2025, time.February, 1), "USDEUR", EUR(0.9)),
	)

	report, err := NewHoldingReport(ledger, NewDate(2025, time.February, 5))
	if err != nil {
		t.Fatalf("NewHoldingReport() error = %v", err)
	}

	wantSecurities := []SecurityHolding{
		{Ticker: "AAPL", ID: "US0378331005.XNAS", Quantity: Q(75), Price: USD(160), MarketValue: EUR(10800)},
		{Ticker: "GOOG", ID: "US38259P5089.XNAS", Quantity: Q(50), Price: USD(2900), MarketValue: EUR(130500)},
	}

	wantCash := []CashHolding{
		{Currency: "EUR", Balance: EUR(10000), Value: EUR(10000)},
		{Currency: "USD", Balance: USD(-151000), Value: EUR(-135900)},
	}

	sort.Slice(report.Securities, func(i, j int) bool {
		return report.Securities[i].Ticker < report.Securities[j].Ticker
	})
	sort.Slice(wantSecurities, func(i, j int) bool {
		return wantSecurities[i].Ticker < wantSecurities[j].Ticker
	})

	if len(report.Securities) != len(wantSecurities) {
		t.Errorf("len(report.Securities) = %d, want %d", len(report.Securities), len(wantSecurities))
	} else {
		for i := range wantSecurities {
			if report.Securities[i].Ticker != wantSecurities[i].Ticker {
				t.Errorf("Ticker[%d] = %s, want %s", i, report.Securities[i].Ticker, wantSecurities[i].Ticker)
			}
			if report.Securities[i].ID != wantSecurities[i].ID {
				t.Errorf("ID[%d] = %s, want %s", i, report.Securities[i].ID, wantSecurities[i].ID)
			}
			if !report.Securities[i].Quantity.Equal(wantSecurities[i].Quantity) {
				t.Errorf("Quantity[%d] = %v, want %v", i, report.Securities[i].Quantity, wantSecurities[i].Quantity)
			}
			if !report.Securities[i].Price.Equal(wantSecurities[i].Price) {
				t.Errorf("Price[%d] = %v, want %v", i, report.Securities[i].Price, wantSecurities[i].Price)
			}
			if !report.Securities[i].MarketValue.Equal(wantSecurities[i].MarketValue) {
				t.Errorf("MarketValue[%d] = %v, want %v", i, report.Securities[i].MarketValue, wantSecurities[i].MarketValue)
			}
		}
	}

	sort.Slice(report.Cash, func(i, j int) bool {
		return report.Cash[i].Currency < report.Cash[j].Currency
	})
	sort.Slice(wantCash, func(i, j int) bool {
		return wantCash[i].Currency < wantCash[j].Currency
	})

	if len(report.Cash) != len(wantCash) {
		t.Errorf("len(report.Cash) = %d, want %d", len(report.Cash), len(wantCash))
	}
	for i := range wantCash {
		if !report.Cash[i].Equals(wantCash[i]) {
			t.Errorf("NewHoldingReport().Cash = %v, want %v", report.Cash[i], wantCash[i])
		}
	}

	wantTotalValue := EUR(15400.0)
	if !report.TotalValue.Equal(wantTotalValue) {
		t.Errorf("NewHoldingReport().TotalValue = %v, want %v", report.TotalValue, wantTotalValue)
	}
}
