package portfolio

import (
	"reflect"
	"testing"

	"github.com/etnz/portfolio/date"
)

func TestLedger_Position(t *testing.T) {
	ledger := &Ledger{
		transactions: []Transaction{
			NewBuy(date.MustParse("2025-01-10"), "", "AAPL", 100, 150.0),
			NewBuy(date.MustParse("2025-01-15"), "", "GOOG", 50, 2800.0),
			NewSell(date.MustParse("2025-02-01"), "", "AAPL", 25, 160.0),
			NewDeposit(date.MustParse("2025-02-05"), "", "USD", 10000), // Should be ignored
			NewBuy(date.MustParse("2025-02-10"), "", "AAPL", 10, 155.0),
			NewSell(date.MustParse("2025-03-01"), "", "GOOG", 50, 2900.0), // Sell all GOOG
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
	tx1_aapl_buy := NewBuy(date.MustParse("2025-01-10"), "", "AAPL", 10, 150.0)
	tx2_aapl_sell := NewSell(date.MustParse("2025-01-15"), "", "AAPL", 5, 155.0)
	tx3_goog_buy := NewBuy(date.MustParse("2025-01-15"), "", "GOOG", 2, 2800.0)
	tx4_aapl_div := NewDividend(date.MustParse("2025-01-20"), "", "AAPL", 20.0)
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
			for _, tx := range ledger.SecurityTransactions(tc.ticker, max) {
				gotTx = append(gotTx, tx)
			}

			if !reflect.DeepEqual(gotTx, tc.wantTx) {
				t.Errorf("SecurityTransactions(%q, %s) got %v, want %v", tc.ticker, tc.maxDate, gotTx, tc.wantTx)
			}
		})
	}
}
