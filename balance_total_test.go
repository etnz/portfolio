package portfolio

import (
	"testing"
	"time"
)

func TestBalance_TotalPortfolioValue(t *testing.T) {
	ledger := NewLedger()
	market := NewMarketData()
	as, err := NewAccountingSystem(ledger, market, "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}
	o := NewDate(2025, time.January, 1)
	apple := NewSecurity(AAPL, "AAPL", "USD")
	google := NewSecurity(GOOG, "GOOG", "USD")
	eurusd := NewSecurity(USDEUR, "USDEUR", "EUR")
	ledger.Append(
		NewDeclare(o, "", apple.Ticker(), apple.ID(), apple.Currency()), NewDeclare(o, "", google.Ticker(), google.ID(), google.Currency()), NewAccrue(o, "interest", "bux", EUR(10.0)),
		NewDeposit(NewDate(2025, time.January, 5), "", EUR(10.0), "bux"),
		NewDeposit(NewDate(2025, time.January, 5), "", EUR(10000), ""),
		NewDeposit(NewDate(2025, time.January, 10), "", USD(50000), ""),
		NewBuy(NewDate(2025, time.January, 10), "", "AAPL", Q(100), USD(15000.0)),
		NewBuy(NewDate(2025, time.January, 15), "", "GOOG", Q(50), USD(14000.0)),
	)
	market.Add(apple)
	market.Add(google)
	market.Add(eurusd)
	market.Append(apple.ID(), NewDate(2025, time.January, 10), 160.0)
	market.Append(google.ID(), NewDate(2025, time.January, 15), 170.0)
	market.Append(eurusd.ID(), NewDate(2025, time.January, 1), 1.1)
	market.Append(eurusd.ID(), NewDate(2025, time.January, 5), 1.15)
	market.Append(eurusd.ID(), NewDate(2025, time.January, 15), 1.2)

	journal, err := as.newJournal()
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name                  string
		date                  Date
		wantTotalMarketValue  Money
		wantTotalCash         Money
		wantTotalCounterparty Money
		wantTotalPortfolio    Money
	}{
		{
			name:                  "On the day of the second buy",
			date:                  NewDate(2025, time.January, 15),
			wantTotalMarketValue:  USD(24500), // (100 * 160) + (50 * 170)
			wantTotalCash:         USD(33012), // 50000 - 15000 - 14000 + (10000+ 10) * 1.2
			wantTotalCounterparty: USD(0),
			wantTotalPortfolio:    USD(57512),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			balance.forex["EUR"] = USD(1.2)

			gotTotalMarketValue := balance.TotalMarketValue()
			if !gotTotalMarketValue.Equal(tc.wantTotalMarketValue) {
				t.Errorf("TotalMarketValue() = %v, want %v", gotTotalMarketValue, tc.wantTotalMarketValue)
			}

			gotTotalCash := balance.TotalCash()
			if !gotTotalCash.Equal(tc.wantTotalCash) {
				t.Errorf("TotalCash() = %v, want %v", gotTotalCash, tc.wantTotalCash)
			}

			gotTotalCounterparty := balance.TotalCounterparty()
			if !gotTotalCounterparty.Equal(tc.wantTotalCounterparty) {
				t.Errorf("TotalCounterparty() = %v, want %v", gotTotalCounterparty, tc.wantTotalCounterparty)
			}

			gotTotalPortfolio := balance.TotalPortfolioValue()
			if !gotTotalPortfolio.Equal(tc.wantTotalPortfolio) {
				t.Errorf("TotalPortfolioValue() = %v, want %v", gotTotalPortfolio, tc.wantTotalPortfolio)
			}
		})
	}
}
