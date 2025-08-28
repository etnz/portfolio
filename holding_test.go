package portfolio

import (
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
)

func TestAccountingSystem_NewHoldingReport(t *testing.T) {
	ledger := NewLedger()
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "USD"),
		NewDeclaration(o, "", "GOOG", "US38259P5089.XNAS", "USD"),
		NewBuy(date.New(2025, time.January, 10), "", "AAPL", 100, 100*150.0),
		NewBuy(date.New(2025, time.January, 15), "", "GOOG", 50, 50*2800.0),
		NewSell(date.New(2025, time.February, 1), "", "AAPL", 25, 25*160.0),
		NewDeposit(date.New(2025, time.February, 5), "", "EUR", 10000),
	)

	market := NewMarketData()
	market.Add(NewSecurity(must(NewMSSI("US0378331005", "XNAS")), "AAPL", "USD"))
	market.Add(NewSecurity(must(NewMSSI("US38259P5089", "XNAS")), "GOOG", "USD"))
	market.Add(NewSecurity(must(NewCurrencyPair("USD", "EUR")), "USDEUR", "EUR"))
	market.Append(must(NewMSSI("US0378331005", "XNAS")), date.New(2025, time.February, 1), 160.0)
	market.Append(must(NewMSSI("US38259P5089", "XNAS")), date.New(2025, time.February, 1), 2900.0)
	market.Append(must(NewCurrencyPair("USD", "EUR")), date.New(2025, time.February, 1), 0.9)

	as, err := NewAccountingSystem(ledger, market, "EUR")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}

	report, err := as.NewHoldingReport(date.New(2025, time.February, 5))
	if err != nil {
		t.Fatalf("NewHoldingReport() error = %v", err)
	}

	wantSecurities := []SecurityHolding{
		{Ticker: "AAPL", ID: "US0378331005.XNAS", Quantity: 75, Price: 160, MarketValue: 10800},
		{Ticker: "GOOG", ID: "US38259P5089.XNAS", Quantity: 50, Price: 2900, MarketValue: 130500},
	}

	wantCash := []CashHolding{
		{Currency: "EUR", Balance: 10000, Value: 10000},
		{Currency: "USD", Balance: -151000, Value: -135900},
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
				t.Errorf("Ticker = %s, want %s", report.Securities[i].Ticker, wantSecurities[i].Ticker)
			}
			if report.Securities[i].ID != wantSecurities[i].ID {
				t.Errorf("ID = %s, want %s", report.Securities[i].ID, wantSecurities[i].ID)
			}
			if report.Securities[i].Quantity != wantSecurities[i].Quantity {
				t.Errorf("Quantity = %f, want %f", report.Securities[i].Quantity, wantSecurities[i].Quantity)
			}
			if report.Securities[i].Price != wantSecurities[i].Price {
				t.Errorf("Price = %f, want %f", report.Securities[i].Price, wantSecurities[i].Price)
			}
			if report.Securities[i].MarketValue != wantSecurities[i].MarketValue {
				t.Errorf("MarketValue = %f, want %f", report.Securities[i].MarketValue, wantSecurities[i].MarketValue)
			}
		}
	}

	sort.Slice(report.Cash, func(i, j int) bool {
		return report.Cash[i].Currency < report.Cash[j].Currency
	})
	sort.Slice(wantCash, func(i, j int) bool {
		return wantCash[i].Currency < wantCash[j].Currency
	})

	if !reflect.DeepEqual(report.Cash, wantCash) {
		t.Errorf("NewHoldingReport().Cash = %v, want %v", report.Cash, wantCash)
	}

	wantTotalValue := 15400.0
	if report.TotalValue != wantTotalValue {
		t.Errorf("NewHoldingReport().TotalValue = %v, want %v", report.TotalValue, wantTotalValue)
	}
}

func must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}
