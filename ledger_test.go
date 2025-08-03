package portfolio

import (
	"reflect"
	"testing"

	"github.com/etnz/portfolio/date"
)

func TestLedger_Position(t *testing.T) {
	ledger := &Ledger{
		transactions: []Transaction{
			NewBuy(date.MustParse("2025-01-10"), "", "AAPL", 100, 150.0, "EUR"),
			NewBuy(date.MustParse("2025-01-15"), "", "GOOG", 50, 2800.0, "EUR"),
			NewSell(date.MustParse("2025-02-01"), "", "AAPL", 25, 160.0, "EUR"),
			NewDeposit(date.MustParse("2025-02-05"), "", "USD", 10000), // Should be ignored
			NewBuy(date.MustParse("2025-02-10"), "", "AAPL", 10, 155.0, "EUR"),
			NewSell(date.MustParse("2025-03-01"), "", "GOOG", 50, 2900.0, "EUR"), // Sell all GOOG
		},
	}
	// The ledger is intentionally created with sorted transactions, as the underlying
	// SecurityTransactions method relies on a sorted list for efficiency.

	testCases := []struct {
		name         string
		ticker       string
		date         string
		wantPosition float64
	}{
		{
			name:         "Before any transactions",
			ticker:       "AAPL",
			date:         "2025-01-09",
			wantPosition: 0,
		},
		{
			name:         "On the day of the first buy",
			ticker:       "AAPL",
			date:         "2025-01-10",
			wantPosition: 100,
		},
		{
			name:         "After first buy, before sell",
			ticker:       "AAPL",
			date:         "2025-01-31",
			wantPosition: 100,
		},
		{
			name:         "On the day of the sell",
			ticker:       "AAPL",
			date:         "2025-02-01",
			wantPosition: 75, // 100 - 25
		},
		{
			name:         "After sell, before second buy",
			ticker:       "AAPL",
			date:         "2025-02-09",
			wantPosition: 75,
		},
		{
			name:         "On the day of the second buy",
			ticker:       "AAPL",
			date:         "2025-02-10",
			wantPosition: 85, // 75 + 10
		},
		{
			name:         "Final position for AAPL",
			ticker:       "AAPL",
			date:         "2025-04-01",
			wantPosition: 85,
		},
		{
			name:         "GOOG position after buy",
			ticker:       "GOOG",
			date:         "2025-01-20",
			wantPosition: 50,
		},
		{
			name:         "GOOG position on sell day",
			ticker:       "GOOG",
			date:         "2025-03-01",
			wantPosition: 0, // 50 - 50
		},
		{
			name:         "GOOG position after selling all",
			ticker:       "GOOG",
			date:         "2025-04-01",
			wantPosition: 0,
		},
		{
			name:         "Position for a ticker with no transactions",
			ticker:       "MSFT",
			date:         "2025-04-01",
			wantPosition: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			day := date.MustParse(tc.date)
			gotPosition := ledger.Position(tc.ticker, day)
			if gotPosition != tc.wantPosition {
				t.Errorf("Position(%q, %s) = %v, want %v", tc.ticker, tc.date, gotPosition, tc.wantPosition)
			}
		})
	}
}

func TestLedger_SecurityTransactions(t *testing.T) {
	// 1. Arrange: Create a sorted ledger with a mix of transactions.
	tx1_aapl_buy := NewBuy(date.MustParse("2025-01-10"), "", "AAPL", 10, 150.0, "EUR")
	tx2_aapl_sell := NewSell(date.MustParse("2025-01-15"), "", "AAPL", 5, 155.0, "EUR")
	tx3_goog_buy := NewBuy(date.MustParse("2025-01-15"), "", "GOOG", 2, 2800.0, "EUR")
	tx4_aapl_div := NewDividend(date.MustParse("2025-01-20"), "", "AAPL", 20.0, "EUR")
	tx5_deposit := NewDeposit(date.MustParse("2025-01-22"), "", "USD", 1000.0) // Should be ignored by SecurityTransactions

	ledger := &Ledger{
		transactions: []Transaction{
			tx1_aapl_buy,
			tx2_aapl_sell,
			tx3_goog_buy,
			tx4_aapl_div,
			tx5_deposit,
		},
	}
	// The ledger is pre-sorted by date for this test.

	testCases := []struct {
		name    string
		ticker  string
		maxDate string
		wantTx  []Transaction
	}{
		{
			name:    "AAPL before any transactions",
			ticker:  "AAPL",
			maxDate: "2025-01-1",
			wantTx:  []Transaction{},
		},
		{
			name:    "AAPL day after first buy",
			ticker:  "AAPL",
			maxDate: "2025-01-10",
			wantTx:  []Transaction{tx1_aapl_buy},
		},
		{
			name:    "AAPL on day before second transaction",
			ticker:  "AAPL",
			maxDate: "2025-01-14",
			wantTx:  []Transaction{tx1_aapl_buy},
		},
		{
			name:    "AAPL on day of second transaction",
			ticker:  "AAPL",
			maxDate: "2025-01-15",
			wantTx:  []Transaction{tx1_aapl_buy, tx2_aapl_sell},
		},
		{
			name:    "AAPL after all its transactions",
			ticker:  "AAPL",
			maxDate: "2025-01-21",
			wantTx:  []Transaction{tx1_aapl_buy, tx2_aapl_sell, tx4_aapl_div},
		},
		{
			name:    "GOOG on day of its transaction",
			ticker:  "GOOG",
			maxDate: "2025-01-15",
			wantTx:  []Transaction{tx3_goog_buy},
		},
		{
			name:    "GOOG before its transaction",
			ticker:  "GOOG",
			maxDate: "2025-01-14",
			wantTx:  []Transaction{},
		},
		{
			name:    "Ticker with no transactions",
			ticker:  "MSFT",
			maxDate: "2025-02-01",
			wantTx:  []Transaction{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			max := date.MustParse(tc.maxDate)
			gotTx := []Transaction{}
			seq := ledger.SecurityTransactions(tc.ticker, max)
			seq(func(_ int, tx Transaction) bool {
				gotTx = append(gotTx, tx)
				return true
			})

			if !reflect.DeepEqual(gotTx, tc.wantTx) {
				t.Errorf("SecurityTransactions(%q, %s) got %v, want %v", tc.ticker, tc.maxDate, gotTx, tc.wantTx)
			}
		})
	}
}

func TestLedger_CashBalance(t *testing.T) {
	ledger := &Ledger{
		transactions: []Transaction{
			// Transactions are sorted by date to match the function's assumption for optimization.
			NewDeposit(date.MustParse("2025-01-05"), "", "EUR", 10000),
			NewDeposit(date.MustParse("2025-01-10"), "", "USD", 50000),
			NewBuy(date.MustParse("2025-01-15"), "", "AAPL", 100, 150.0, "USD"), // -15000 USD
			NewSell(date.MustParse("2025-02-01"), "", "AAPL", 25, 160.0, "USD"),  // +4000 USD
			NewDividend(date.MustParse("2025-02-15"), "", "AAPL", 75, "USD"),     // +75 USD
			NewWithdraw(date.MustParse("2025-03-01"), "", "USD", 1000),           // -1000 USD
			NewConvert(date.MustParse("2025-03-10"), "", "USD", 2000, "EUR", 1800), // -2000 USD, +1800 EUR
			NewWithdraw(date.MustParse("2025-04-01"), "", "EUR", 500),            // -500 EUR
		},
	}

	testCases := []struct {
		name        string
		currency    string
		date        string
		wantBalance float64
	}{
		// USD Balance Checks
		{
			name:        "USD before any transactions",
			currency:    "USD",
			date:        "2025-01-09",
			wantBalance: 0,
		},
		{
			name:        "USD after deposit",
			currency:    "USD",
			date:        "2025-01-10",
			wantBalance: 50000,
		},
		{
			name:        "USD after buy",
			currency:    "USD",
			date:        "2025-01-15",
			wantBalance: 35000, // 50000 - (100 * 150)
		},
		{
			name:        "USD after sell",
			currency:    "USD",
			date:        "2025-02-01",
			wantBalance: 39000, // 35000 + (25 * 160)
		},
		{
			name:        "USD after dividend",
			currency:    "USD",
			date:        "2025-02-15",
			wantBalance: 39075, // 39000 + 75
		},
		{
			name:        "USD after withdraw",
			currency:    "USD",
			date:        "2025-03-01",
			wantBalance: 38075, // 39075 - 1000
		},
		{
			name:        "USD final balance after convert",
			currency:    "USD",
			date:        "2025-04-01",
			wantBalance: 36075, // 38075 - 2000
		},
		// EUR Balance Checks
		{
			name:        "EUR after deposit",
			currency:    "EUR",
			date:        "2025-01-05",
			wantBalance: 10000,
		},
		{
			name:        "EUR before convert",
			currency:    "EUR",
			date:        "2025-03-09",
			wantBalance: 10000,
		},
		{
			name:        "EUR on convert date",
			currency:    "EUR",
			date:        "2025-03-10",
			wantBalance: 11800, // 10000 + 1800
		},
		{
			name:        "EUR final balance after withdraw",
			currency:    "EUR",
			date:        "2025-05-01",
			wantBalance: 11300, // 11800 - 500
		},
		// Other
		{
			name:        "Balance for currency with no transactions",
			currency:    "GBP",
			date:        "2025-05-01",
			wantBalance: 0,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			day := date.MustParse(tc.date)
			gotBalance := ledger.CashBalance(tc.currency, day)
			if gotBalance != tc.wantBalance {
				t.Errorf("CashBalance(%q, %s) = %v, want %v", tc.currency, tc.date, gotBalance, tc.wantBalance)
			}
		})
	}
}
