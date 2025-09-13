package portfolio

import (
	"testing"
	"time"
)

func TestBalance_Position(t *testing.T) {
	ledger := NewLedger()
	o := NewDate(2025, time.January, 1)
	ledger.Append(
		NewDeclare(o, "", "AAPL", AAPL, "USD"),
		NewDeclare(o, "", "GOOG", GOOG, "USD"),
		NewBuy(NewDate(2025, time.January, 10), "", "AAPL", Q(100), USD(100*150.0)),
		NewBuy(NewDate(2025, time.January, 15), "", "GOOG", Q(50), USD(50*2800.0)),
		NewSell(NewDate(2025, time.February, 1), "", "AAPL", Q(25), USD(25*160.0)),
		NewDeposit(NewDate(2025, time.February, 5), "", USD(10000), ""), // Should be ignored
		NewBuy(NewDate(2025, time.February, 10), "", "AAPL", Q(10), USD(10*155.0)),
		NewSell(NewDate(2025, time.March, 1), "", "GOOG", Q(50), USD(50*2900.0)), // Sell all GOOG
	)

	journal, err := newJournal(ledger, "USD")
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name         string
		ticker       string
		date         Date
		wantPosition Quantity
	}{
		{
			name:         "Before any transactions",
			ticker:       "AAPL",
			date:         NewDate(2025, time.January, 9),
			wantPosition: Q(0),
		},
		{
			name:         "On the day of the first buy",
			ticker:       "AAPL",
			date:         NewDate(2025, time.January, 10),
			wantPosition: Q(100),
		},
		{
			name:         "After first buy, before sell",
			ticker:       "AAPL",
			date:         NewDate(2025, time.January, 31),
			wantPosition: Q(100),
		},
		{
			name:         "On the day of the sell",
			ticker:       "AAPL",
			date:         NewDate(2025, time.February, 1),
			wantPosition: Q(75), // 100 - 25
		},
		{
			name:         "After sell, before second buy",
			ticker:       "AAPL",
			date:         NewDate(2025, time.February, 9),
			wantPosition: Q(75),
		},
		{
			name:         "On the day of the second buy",
			ticker:       "AAPL",
			date:         NewDate(2025, time.February, 10),
			wantPosition: Q(85), // 75 + 10
		},
		{
			name:         "Final position for AAPL",
			ticker:       "AAPL",
			date:         NewDate(2025, time.April, 1),
			wantPosition: Q(85),
		},
		{
			name:         "GOOG position after buy",
			ticker:       "GOOG",
			date:         NewDate(2025, time.January, 20),
			wantPosition: Q(50),
		},
		{
			name:         "GOOG position on sell day",
			ticker:       "GOOG",
			date:         NewDate(2025, time.March, 1),
			wantPosition: Q(0), // 50 - 50
		},
		{
			name:         "GOOG position after selling all",
			ticker:       "GOOG",
			date:         NewDate(2025, time.April, 1),
			wantPosition: Q(0),
		},
		{
			name:         "Position for a ticker with no transactions",
			ticker:       "MSFT",
			date:         NewDate(2025, time.April, 1),
			wantPosition: Q(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotPosition := balance.Position(tc.ticker)
			if !gotPosition.Equal(tc.wantPosition) {
				t.Errorf("Position(%q, %s) = %v, want %v", tc.ticker, tc.date, gotPosition, tc.wantPosition)
			}
		})
	}
}

func TestBalance_BuysAndSells(t *testing.T) {
	ledger := NewLedger()
	o := NewDate(2025, time.January, 1)
	ledger.Append(
		NewDeclare(o, "", "AAPL", AAPL, "USD"),
		NewDeclare(o, "", "GOOG", GOOG, "USD"),
		NewBuy(NewDate(2025, time.January, 10), "", "AAPL", Q(100), USD(15000)), // Buy 1
		NewBuy(NewDate(2025, time.January, 15), "", "GOOG", Q(50), USD(140000)), // Buy 1
		NewSell(NewDate(2025, time.February, 1), "", "AAPL", Q(25), USD(4000)),  // Sell 1
		NewBuy(NewDate(2025, time.February, 10), "", "AAPL", Q(10), USD(1550)),  // Buy 2
		NewSell(NewDate(2025, time.March, 1), "", "GOOG", Q(50), USD(145000)),   // Sell 1
		NewSell(NewDate(2025, time.March, 15), "", "AAPL", Q(50), USD(8000)),    // Sell 2
	)

	journal, err := newJournal(ledger, "USD")
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name      string
		ticker    string
		date      Date
		wantBuys  Money
		wantSells Money
	}{
		{
			name:      "AAPL before any transactions",
			ticker:    "AAPL",
			date:      NewDate(2025, time.January, 9),
			wantBuys:  USD(0),
			wantSells: USD(0),
		},
		{
			name:      "AAPL after first buy",
			ticker:    "AAPL",
			date:      NewDate(2025, time.January, 10),
			wantBuys:  USD(15000),
			wantSells: USD(0),
		},
		{
			name:      "AAPL after first sell",
			ticker:    "AAPL",
			date:      NewDate(2025, time.February, 1),
			wantBuys:  USD(15000),
			wantSells: USD(4000),
		},
		{
			name:      "AAPL after second buy",
			ticker:    "AAPL",
			date:      NewDate(2025, time.February, 10),
			wantBuys:  USD(16550), // 15000 + 1550
			wantSells: USD(4000),
		},
		{
			name:      "AAPL final state",
			ticker:    "AAPL",
			date:      NewDate(2025, time.April, 1),
			wantBuys:  USD(16550),
			wantSells: USD(12000), // 4000 + 8000
		},
		{
			name:      "GOOG final state",
			ticker:    "GOOG",
			date:      NewDate(2025, time.April, 1),
			wantBuys:  USD(140000),
			wantSells: USD(145000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalance() error = %v", err)
			}
			if gotBuys := balance.Buys(tc.ticker); !gotBuys.Equal(tc.wantBuys) {
				t.Errorf("Buys() = %v, want %v", gotBuys, tc.wantBuys)
			}
			if gotSells := balance.Sells(tc.ticker); !gotSells.Equal(tc.wantSells) {
				t.Errorf("Sells() = %v, want %v", gotSells, tc.wantSells)
			}
		})
	}
}

func TestBalance_CashBalance(t *testing.T) {
	ledger := NewLedger()
	o := NewDate(2025, time.January, 1)
	ledger.Append(
		NewDeclare(o, "", "AAPL", AAPL, "USD"),
		// Transactions are sorted by date to match the function's assumption for optimization.
		NewDeposit(NewDate(2025, time.January, 5), "", EUR(10000), ""),
		NewDeposit(NewDate(2025, time.January, 10), "", USD(50000), ""),
		NewBuy(NewDate(2025, time.January, 15), "", "AAPL", Q(100), USD(100*150.0)), // -15000 USD
		NewSell(NewDate(2025, time.February, 1), "", "AAPL", Q(25), USD(25*160.0)),  // +4000 USD
		NewDividend(NewDate(2025, time.February, 15), "", "AAPL", USD(75)),          // +75 USD
		NewWithdraw(NewDate(2025, time.March, 1), "", USD(1000)),                    // -1000 USD
		NewConvert(NewDate(2025, time.March, 10), "", USD(2000), EUR(1800)),         // -2000 USD, +1800 EUR
		NewWithdraw(NewDate(2025, time.April, 1), "", EUR(500)),                     // -500 EUR
	)

	journal, err := newJournal(ledger, "USD")
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name        string
		currency    string
		date        Date
		wantBalance Money
	}{
		// USD Balance Checks
		{
			name:        "USD before any transactions",
			currency:    "USD",
			date:        NewDate(2025, time.January, 9),
			wantBalance: USD(0),
		},
		{
			name:        "USD after deposit",
			currency:    "USD",
			date:        NewDate(2025, time.January, 10),
			wantBalance: USD(50000),
		},
		{
			name:        "USD after buy",
			currency:    "USD",
			date:        NewDate(2025, time.January, 15),
			wantBalance: USD(35000), // 50000 - (100 * 150)
		},
		{
			name:        "USD after sell",
			currency:    "USD",
			date:        NewDate(2025, time.February, 1),
			wantBalance: USD(39000), // 35000 + (25 * 160)
		},
		{
			name:        "USD after dividend",
			currency:    "USD",
			date:        NewDate(2025, time.February, 15),
			wantBalance: USD(39075), // 39000 + 75
		},
		{
			name:        "USD after withdraw",
			currency:    "USD",
			date:        NewDate(2025, time.March, 1),
			wantBalance: USD(38075), // 39075 - 1000
		},
		{
			name:        "USD final balance after convert",
			currency:    "USD",
			date:        NewDate(2025, time.April, 1),
			wantBalance: USD(36075), // 38075 - 2000
		},
		// EUR Balance Checks
		{
			name:        "EUR after deposit",
			currency:    "EUR",
			date:        NewDate(2025, time.January, 5),
			wantBalance: EUR(10000),
		},
		{
			name:        "EUR before convert",
			currency:    "EUR",
			date:        NewDate(2025, time.March, 9),
			wantBalance: EUR(10000),
		},
		{
			name:        "EUR on convert date",
			currency:    "EUR",
			date:        NewDate(2025, time.March, 10),
			wantBalance: EUR(11800), // 10000 + 1800
		},
		{
			name:        "EUR final balance after withdraw",
			currency:    "EUR",
			date:        NewDate(2025, time.May, 1),
			wantBalance: EUR(11300), // 11800 - 500
		},
		// Other
		{
			name:        "Balance for currency with no transactions",
			currency:    "GBP",
			date:        NewDate(2025, time.May, 1),
			wantBalance: M(0, "GBP"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotBalance := balance.Cash(tc.currency)
			if !gotBalance.Equal(tc.wantBalance) {
				t.Errorf("CashBalance(%q, %s) = %v, want %v", tc.currency, tc.date, gotBalance, tc.wantBalance)
			}
		})
	}
}

func TestBalance_CounterpartyAccountBalance(t *testing.T) {
	ledger := NewLedger()
	o := NewDate(2025, time.January, 1)
	ledger.Append(
		NewAccrue(o, "interest", "bux", EUR(10.0)),
		NewDeposit(NewDate(2025, time.January, 5), "", EUR(10.0), "bux"),
	)

	journal, err := newJournal(ledger, "USD")
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name        string
		account     string
		date        Date
		wantBalance Money
	}{
		{
			name:        "Initial balance",
			account:     "bux",
			date:        o,
			wantBalance: EUR(10.0),
		},
		{
			name:        "Final balance",
			account:     "bux",
			date:        NewDate(2025, time.January, 5),
			wantBalance: EUR(0),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotBalance := balance.Counterparty(tc.account)
			if !gotBalance.Equal(tc.wantBalance) {
				t.Errorf("CounterpartyAccountBalance(%q, %s) = %v, want %v", tc.account, tc.date, gotBalance, tc.wantBalance)
			}
		})
	}
}

func TestBalance_TotalMarketValue(t *testing.T) {
	ledger := NewLedger()
	o := NewDate(2025, time.January, 1)
	apple := NewSecurity(AAPL, "AAPL", "USD")
	ledger.Append(
		NewDeclare(o, "", apple.Ticker(), apple.ID(), apple.Currency()),
		NewBuy(NewDate(2025, time.January, 10), "", "AAPL", Q(100), USD(15000.0)),
		NewUpdatePrice(NewDate(2025, time.January, 10), "AAPL", USD(160.0)),
	)

	journal, err := newJournal(ledger, "USD")
	if err != nil {
		t.Fatalf("NewJournal() error = %v", err)
	}

	testCases := []struct {
		name      string
		ticker    string
		date      Date
		wantValue Money
	}{
		{
			name:      "On the day of the buy",
			ticker:    "AAPL",
			date:      NewDate(2025, time.January, 10),
			wantValue: USD(16000),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			balance, err := NewBalance(journal, tc.date, FIFO)
			if err != nil {
				t.Fatalf("NewBalanceFromJournal() error = %v", err)
			}
			gotValue := balance.MarketValue(tc.ticker)
			if !gotValue.Equal(tc.wantValue) {
				t.Errorf("TotalMarketValue(%q, %s) = %v, want %v", tc.ticker, tc.date, gotValue, tc.wantValue)
			}
		})
	}
}
