package portfolio

import (
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

func TestBalance_TotalPortfolioValue(t *testing.T) {
	ledger := NewLedger()
	market := NewMarketData()
	as, err := NewAccountingSystem(ledger, market, "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}
	o := date.New(2025, time.January, 1)
	apple := NewSecurity("US0378331005.XNAS", "AAPL", "USD")
	google := NewSecurity("US38259P5089.XNAS", "GOOG", "USD")
	eurusd := NewSecurity("EURUSD", "USDEUR", "EUR")
	ledger.Append(
		NewDeclaration(o, "", apple.Ticker(), apple.ID().String(), apple.Currency()),
		NewDeclaration(o, "", google.Ticker(), google.ID().String(), google.Currency()),
		NewCreatedAccrue(o, "interest", "bux", 10.0, "EUR"),
		NewDeposit(date.New(2025, time.January, 5), "", "EUR", 10.0, "bux"),
		NewDeposit(date.New(2025, time.January, 5), "", "EUR", 10000, ""),
		NewDeposit(date.New(2025, time.January, 10), "", "USD", 50000, ""),
		NewBuy(date.New(2025, time.January, 10), "", "AAPL", 100, 15000.0),
		NewBuy(date.New(2025, time.January, 15), "", "GOOG", 50, 14000.0),
	)
	market.Add(apple)
	market.Add(google)
	market.Add(eurusd)
	market.Append(apple.ID(), date.New(2025, time.January, 10), 160.0)
	market.Append(google.ID(), date.New(2025, time.January, 15), 170.0)
	market.Append(eurusd.ID(), date.New(2025, time.January, 1), 1.1)
	market.Append(eurusd.ID(), date.New(2025, time.January, 5), 1.15)
	market.Append(eurusd.ID(), date.New(2025, time.January, 15), 1.2)

	journal, err := as.newJournal()
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name                  string
		date                  date.Date
		wantTotalMarketValue  float64
		wantTotalCash         float64
		wantTotalCounterparty float64
		wantTotalPortfolio    float64
	}{
		{
			name:                  "On the day of the second buy",
			date:                  date.New(2025, time.January, 15),
			wantTotalMarketValue:  24500, // (100 * 160) + (50 * 170)
			wantTotalCash:         33012, // 50000 - 15000 - 14000 + (10000+ 10) * 1.2
			wantTotalCounterparty: 0,
			wantTotalPortfolio:    57512,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalanceFromJournal(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			balance.forex["EUR"] = decimal.NewFromFloat(1.2)

			gotTotalMarketValue := balance.TotalMarketValue()
			wantTotalMarketValue := decimal.NewFromFloat(tc.wantTotalMarketValue)
			if !gotTotalMarketValue.Equal(wantTotalMarketValue) {
				t.Errorf("TotalMarketValue() = %v, want %v", gotTotalMarketValue, wantTotalMarketValue)
			}

			gotTotalCash := balance.TotalCash()
			wantTotalCash := decimal.NewFromFloat(tc.wantTotalCash)
			if !gotTotalCash.Equal(wantTotalCash) {
				t.Errorf("TotalCash() = %v, want %v", gotTotalCash, wantTotalCash)
			}

			gotTotalCounterparty := balance.TotalCounterparty()
			wantTotalCounterparty := decimal.NewFromFloat(tc.wantTotalCounterparty)
			if !gotTotalCounterparty.Equal(wantTotalCounterparty) {
				t.Errorf("TotalCounterparty() = %v, want %v", gotTotalCounterparty, wantTotalCounterparty)
			}

			gotTotalPortfolio := balance.TotalPortfolioValue()
			wantTotalPortfolio := decimal.NewFromFloat(tc.wantTotalPortfolio)
			if !gotTotalPortfolio.Equal(wantTotalPortfolio) {
				t.Errorf("TotalPortfolioValue() = %v, want %v", gotTotalPortfolio, wantTotalPortfolio)
			}
		})
	}
}
