package portfolio

import (
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
	"github.com/shopspring/decimal"
)

func TestBalance_Position(t *testing.T) {
	ledger := NewLedger()
	as, err := NewAccountingSystem(ledger, NewMarketData(), "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "USD"),
		NewDeclaration(o, "", "GOOG", "US38259P5089.XNAS", "USD"),
		NewBuy(date.New(2025, time.January, 10), "", "AAPL", 100, 100*150.0),
		NewBuy(date.New(2025, time.January, 15), "", "GOOG", 50, 50*2800.0),
		NewSell(date.New(2025, time.February, 1), "", "AAPL", 25, 25*160.0),
		NewDeposit(date.New(2025, time.February, 5), "", "USD", 10000, ""), // Should be ignored
		NewBuy(date.New(2025, time.February, 10), "", "AAPL", 10, 10*155.0),
		NewSell(date.New(2025, time.March, 1), "", "GOOG", 50, 50*2900.0), // Sell all GOOG
	)

	journal, err := as.newJournal()
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name         string
		ticker       string
		date         date.Date
		wantPosition float64
	}{
		{
			name:         "Before any transactions",
			ticker:       "AAPL",
			date:         date.New(2025, time.January, 9),
			wantPosition: 0,
		},
		{
			name:         "On the day of the first buy",
			ticker:       "AAPL",
			date:         date.New(2025, time.January, 10),
			wantPosition: 100,
		},
		{
			name:         "After first buy, before sell",
			ticker:       "AAPL",
			date:         date.New(2025, time.January, 31),
			wantPosition: 100,
		},
		{
			name:         "On the day of the sell",
			ticker:       "AAPL",
			date:         date.New(2025, time.February, 1),
			wantPosition: 75, // 100 - 25
		},
		{
			name:         "After sell, before second buy",
			ticker:       "AAPL",
			date:         date.New(2025, time.February, 9),
			wantPosition: 75,
		},
		{
			name:         "On the day of the second buy",
			ticker:       "AAPL",
			date:         date.New(2025, time.February, 10),
			wantPosition: 85, // 75 + 10
		},
		{
			name:         "Final position for AAPL",
			ticker:       "AAPL",
			date:         date.New(2025, time.April, 1),
			wantPosition: 85,
		},
		{
			name:         "GOOG position after buy",
			ticker:       "GOOG",
			date:         date.New(2025, time.January, 20),
			wantPosition: 50,
		},
		{
			name:         "GOOG position on sell day",
			ticker:       "GOOG",
			date:         date.New(2025, time.March, 1),
			wantPosition: 0, // 50 - 50
		},
		{
			name:         "GOOG position after selling all",
			ticker:       "GOOG",
			date:         date.New(2025, time.April, 1),
			wantPosition: 0,
		},
		{
			name:         "Position for a ticker with no transactions",
			ticker:       "MSFT",
			date:         date.New(2025, time.April, 1),
			wantPosition: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotPosition := balance.Position(tc.ticker)
			wantPosition := decimal.NewFromFloat(tc.wantPosition)
			if !gotPosition.Equal(wantPosition) {
				t.Errorf("Position(%q, %s) = %v, want %v", tc.ticker, tc.date, gotPosition, wantPosition)
			}
		})
	}
}

func TestBalance_CashBalance(t *testing.T) {
	ledger := NewLedger()
	as, err := NewAccountingSystem(ledger, NewMarketData(), "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "USD"),
		// Transactions are sorted by date to match the function's assumption for optimization.
		NewDeposit(date.New(2025, time.January, 5), "", "EUR", 10000, ""),
		NewDeposit(date.New(2025, time.January, 10), "", "USD", 50000, ""),
		NewBuy(date.New(2025, time.January, 15), "", "AAPL", 100, 100*150.0),     // -15000 USD
		NewSell(date.New(2025, time.February, 1), "", "AAPL", 25, 25*160.0),      // +4000 USD
		NewDividend(date.New(2025, time.February, 15), "", "AAPL", 75),           // +75 USD
		NewWithdraw(date.New(2025, time.March, 1), "", "USD", 1000),              // -1000 USD
		NewConvert(date.New(2025, time.March, 10), "", "USD", 2000, "EUR", 1800), // -2000 USD, +1800 EUR
		NewWithdraw(date.New(2025, time.April, 1), "", "EUR", 500),               // -500 EUR
	)

	journal, err := as.newJournal()
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name        string
		currency    string
		date        date.Date
		wantBalance float64
	}{
		// USD Balance Checks
		{
			name:        "USD before any transactions",
			currency:    "USD",
			date:        date.New(2025, time.January, 9),
			wantBalance: 0,
		},
		{
			name:        "USD after deposit",
			currency:    "USD",
			date:        date.New(2025, time.January, 10),
			wantBalance: 50000,
		},
		{
			name:        "USD after buy",
			currency:    "USD",
			date:        date.New(2025, time.January, 15),
			wantBalance: 35000, // 50000 - (100 * 150)
		},
		{
			name:        "USD after sell",
			currency:    "USD",
			date:        date.New(2025, time.February, 1),
			wantBalance: 39000, // 35000 + (25 * 160)
		},
		{
			name:        "USD after dividend",
			currency:    "USD",
			date:        date.New(2025, time.February, 15),
			wantBalance: 39075, // 39000 + 75
		},
		{
			name:        "USD after withdraw",
			currency:    "USD",
			date:        date.New(2025, time.March, 1),
			wantBalance: 38075, // 39075 - 1000
		},
		{
			name:        "USD final balance after convert",
			currency:    "USD",
			date:        date.New(2025, time.April, 1),
			wantBalance: 36075, // 38075 - 2000
		},
		// EUR Balance Checks
		{
			name:        "EUR after deposit",
			currency:    "EUR",
			date:        date.New(2025, time.January, 5),
			wantBalance: 10000,
		},
		{
			name:        "EUR before convert",
			currency:    "EUR",
			date:        date.New(2025, time.March, 9),
			wantBalance: 10000,
		},
		{
			name:        "EUR on convert date",
			currency:    "EUR",
			date:        date.New(2025, time.March, 10),
			wantBalance: 11800, // 10000 + 1800
		},
		{
			name:        "EUR final balance after withdraw",
			currency:    "EUR",
			date:        date.New(2025, time.May, 1),
			wantBalance: 11300, // 11800 - 500
		},
		// Other
		{
			name:        "Balance for currency with no transactions",
			currency:    "GBP",
			date:        date.New(2025, time.May, 1),
			wantBalance: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotBalance := balance.Cash(tc.currency)
			wantBalance := decimal.NewFromFloat(tc.wantBalance)
			if !gotBalance.Equal(wantBalance) {
				t.Errorf("CashBalance(%q, %s) = %v, want %v", tc.currency, tc.date, gotBalance, wantBalance)
			}
		})
	}
}

func TestBalance_CounterpartyAccountBalance(t *testing.T) {
	ledger := NewLedger()
	as, err := NewAccountingSystem(ledger, NewMarketData(), "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewAccrue(o, "interest", "bux", 10.0, "EUR"),
		NewDeposit(date.New(2025, time.January, 5), "", "EUR", 10.0, "bux"),
	)

	journal, err := as.newJournal()
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name        string
		account     string
		date        date.Date
		wantBalance float64
	}{
		{
			name:        "Initial balance",
			account:     "bux",
			date:        o,
			wantBalance: 10.0,
		},
		{
			name:        "Final balance",
			account:     "bux",
			date:        date.New(2025, time.January, 5),
			wantBalance: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotBalance := balance.Counterparty(tc.account)
			wantBalance := decimal.NewFromFloat(tc.wantBalance)
			if !gotBalance.Equal(wantBalance) {
				t.Errorf("CounterpartyAccountBalance(%q, %s) = %v, want %v", tc.account, tc.date, gotBalance, wantBalance)
			}
		})
	}
}

func TestBalance_TotalMarketValue(t *testing.T) {
	ledger := NewLedger()
	market := NewMarketData()
	as, err := NewAccountingSystem(ledger, market, "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}
	o := date.New(2025, time.January, 1)
	apple := NewSecurity("US0378331005.XNAS", "AAPL", "USD")
	ledger.Append(
		NewDeclaration(o, "", apple.Ticker(), apple.ID().String(), apple.Currency()),
		NewBuy(date.New(2025, time.January, 10), "", "AAPL", 100, 15000.0),
	)
	market.Add(apple)
	market.Append(NewSecurity("US0378331005.XNAS", "AAPL", "USD").ID(), date.New(2025, time.January, 10), 160.0)

	journal, err := as.newJournal()
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name      string
		ticker    string
		date      date.Date
		wantValue float64
	}{
		{
			name:      "On the day of the buy",
			ticker:    "AAPL",
			date:      date.New(2025, time.January, 10),
			wantValue: 16000,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotValue := balance.MarketValue(tc.ticker)
			wantValue := decimal.NewFromFloat(tc.wantValue)
			if !gotValue.Equal(wantValue) {
				t.Errorf("TotalMarketValue(%q, %s) = %v, want %v", tc.ticker, tc.date, gotValue, wantValue)
			}
		})
	}
}
