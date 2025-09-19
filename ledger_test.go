package portfolio

import (
	"testing"
	"time"
)

func TestLedger_CashBalance(t *testing.T) {
	ledger := NewLedger()
	o := NewDate(2025, time.January, 1)
	ledger.Append(
		NewDeclare(o, "", "AAPL", AAPL, "USD"),
		// Transactions are sorted by date to match the function's assumption for optimization.
		NewDeposit(NewDate(2025, time.January, 5), "", EUR(10000), ""),
		NewDeposit(NewDate(2025, time.January, 10), "", USD(50000), ""),             // +50000 USD
		NewBuy(NewDate(2025, time.January, 15), "", "AAPL", Q(100), USD(100*150.0)), // -15000 USD
		NewSell(NewDate(2025, time.February, 1), "", "AAPL", Q(25), USD(25*160.0)),  // +4000 USD
		NewDividend(NewDate(2025, time.February, 15), "", "AAPL", USD(75)),          // +75 USD
		NewWithdraw(NewDate(2025, time.March, 1), "", USD(1000)),                    // -1000 USD
		NewConvert(NewDate(2025, time.March, 10), "", USD(2000), EUR(1800)),         // -2000 USD, +1800 EUR
		NewWithdraw(NewDate(2025, time.April, 1), "", EUR(500)),                     // -500 EUR
	)

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
			wantBalance: USD(39000), // 39000
		},
		{
			name:        "USD after withdraw",
			currency:    "USD",
			date:        NewDate(2025, time.March, 1),
			wantBalance: USD(38000), // 39000 - 1000
		},
		{
			name:        "USD final balance after convert",
			currency:    "USD",
			date:        NewDate(2025, time.April, 1),
			wantBalance: USD(36000), // 38000 - 2000
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
			date:        NewDate(2025, time.March, 10), // 10000 + 1800
			wantBalance: EUR(11800),
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
			gotBalance := ledger.CashBalance(tc.currency, tc.date)
			if !gotBalance.Equal(tc.wantBalance) {
				t.Errorf("CashBalance(%q, %s) = %v, want %v", tc.currency, tc.date, gotBalance, tc.wantBalance)
			}
		})
	}
}
