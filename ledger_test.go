package portfolio

import (
	"reflect"
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
)

// TODO: add a Position test with some splits

func TestLedger_Position(t *testing.T) {
	ledger := NewLedger()
	market := NewMarketData()
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "EUR"),
		NewDeclaration(o, "", "GOOG", "US38259P5089.XNAS", "EUR"),
		NewBuy(date.New(2025, time.January, 10), "", "AAPL", 100, 100*150.0),
		NewBuy(date.New(2025, time.January, 15), "", "GOOG", 50, 50*2800.0),
		NewSell(date.New(2025, time.February, 1), "", "AAPL", 25, 25*160.0),
		NewDeposit(date.New(2025, time.February, 5), "", "USD", 10000), // Should be ignored
		NewBuy(date.New(2025, time.February, 10), "", "AAPL", 10, 10*155.0),
		NewSell(date.New(2025, time.March, 1), "", "GOOG", 50, 50*2900.0), // Sell all GOOG
	)
	// The ledger is intentionally created with sorted transactions, as the underlying
	// SecurityTransactions method relies on a sorted list for efficiency.

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
			gotPosition := ledger.Position(tc.ticker, tc.date, market)
			if gotPosition != tc.wantPosition {
				t.Errorf("Position(%q, %s) = %v, want %v", tc.ticker, tc.date, gotPosition, tc.wantPosition)
			}
		})
	}
}

func TestLedger_SecurityTransactions(t *testing.T) {
	// 1. Arrange: Create a sorted ledger with a mix of transactions.
	tx1_aapl_buy := NewBuy(date.New(2025, time.January, 10), "", "AAPL", 10, 10*150.0)
	tx2_aapl_sell := NewSell(date.New(2025, time.January, 15), "", "AAPL", 5, 5*155.0)
	tx3_goog_buy := NewBuy(date.New(2025, time.January, 15), "", "GOOG", 2, 2*2800.0)
	tx4_aapl_div := NewDividend(date.New(2025, time.January, 20), "", "AAPL", 20.0)
	tx5_deposit := NewDeposit(date.New(2025, time.January, 22), "", "USD", 1000.0) // Should be ignored by SecurityTransactions

	ledger := NewLedger()
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "EUR"),
		NewDeclaration(o, "", "GOOG", "US38259P5089.XNAS", "EUR"),
		tx1_aapl_buy,
		tx2_aapl_sell,
		tx3_goog_buy,
		tx4_aapl_div,
		tx5_deposit,
	)
	// The ledger is pre-sorted by date for this test.

	testCases := []struct {
		name    string
		ticker  string
		maxDate date.Date
		wantTx  []Transaction
	}{
		{
			name:    "AAPL before any transactions",
			ticker:  "AAPL",
			maxDate: date.New(2025, time.January, 1),
			wantTx:  []Transaction{},
		},
		{
			name:    "AAPL day after first buy",
			ticker:  "AAPL",
			maxDate: date.New(2025, time.January, 10),
			wantTx:  []Transaction{tx1_aapl_buy},
		},
		{
			name:    "AAPL on day before second transaction",
			ticker:  "AAPL",
			maxDate: date.New(2025, time.January, 14),
			wantTx:  []Transaction{tx1_aapl_buy},
		},
		{
			name:    "AAPL on day of second transaction",
			ticker:  "AAPL",
			maxDate: date.New(2025, time.January, 15),
			wantTx:  []Transaction{tx1_aapl_buy, tx2_aapl_sell},
		},
		{
			name:    "AAPL after all its transactions",
			ticker:  "AAPL",
			maxDate: date.New(2025, time.January, 21),
			wantTx:  []Transaction{tx1_aapl_buy, tx2_aapl_sell, tx4_aapl_div},
		},
		{
			name:    "GOOG on day of its transaction",
			ticker:  "GOOG",
			maxDate: date.New(2025, time.January, 15),
			wantTx:  []Transaction{tx3_goog_buy},
		},
		{
			name:    "GOOG before its transaction",
			ticker:  "GOOG",
			maxDate: date.New(2025, time.January, 14),
			wantTx:  []Transaction{},
		},
		{
			name:    "Ticker with no transactions",
			ticker:  "MSFT",
			maxDate: date.New(2025, time.February, 1),
			wantTx:  []Transaction{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotTx := []Transaction{}
			seq := ledger.SecurityTransactions(tc.ticker, tc.maxDate)
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
	ledger := NewLedger()
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "USD"),
		// Transactions are sorted by date to match the function's assumption for optimization.
		NewDeposit(date.New(2025, time.January, 5), "", "EUR", 10000),
		NewDeposit(date.New(2025, time.January, 10), "", "USD", 50000),
		NewBuy(date.New(2025, time.January, 15), "", "AAPL", 100, 100*150.0),     // -15000 USD
		NewSell(date.New(2025, time.February, 1), "", "AAPL", 25, 25*160.0),      // +4000 USD
		NewDividend(date.New(2025, time.February, 15), "", "AAPL", 75),           // +75 USD
		NewWithdraw(date.New(2025, time.March, 1), "", "USD", 1000),              // -1000 USD
		NewConvert(date.New(2025, time.March, 10), "", "USD", 2000, "EUR", 1800), // -2000 USD, +1800 EUR
		NewWithdraw(date.New(2025, time.April, 1), "", "EUR", 500),               // -500 EUR
	)

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
			gotBalance := ledger.CashBalance(tc.currency, tc.date)
			if gotBalance != tc.wantBalance {
				t.Errorf("CashBalance(%q, %s) = %v, want %v", tc.currency, tc.date, gotBalance, tc.wantBalance)
			}
		})
	}
}

func TestLedger_CostBasisAndRealizedGain(t *testing.T) {
	ledger := NewLedger()
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "USD"),
		NewBuy(date.New(2025, time.January, 10), "", "AAPL", 100, 100*150.0), // Cost: 15000
		NewBuy(date.New(2025, time.January, 15), "", "AAPL", 50, 50*160.0),   // Cost: 8000
		NewSell(date.New(2025, time.February, 1), "", "AAPL", 75, 75*170.0),  // Proceeds: 12750
		NewBuy(date.New(2025, time.February, 10), "", "AAPL", 25, 25*180.0),  // Cost: 4500
		NewSell(date.New(2025, time.March, 1), "", "AAPL", 100, 100*190.0),   // Proceeds: 19000
	)

	testCases := []struct {
		name             string
		ticker           string
		date             date.Date
		method           CostBasisMethod
		wantCostBasis    float64
		wantRealizedGain float64
	}{
		{
			name:             "FIFO - After first sell",
			ticker:           "AAPL",
			date:             date.New(2025, time.February, 5),
			method:           FIFO,
			wantCostBasis:    (25 * 150) + (50 * 160),
			wantRealizedGain: (75 * 170) - (75 * 150),
		},
		{
			name:             "FIFO - Final",
			ticker:           "AAPL",
			date:             date.New(2025, time.April, 1),
			method:           FIFO,
			wantCostBasis:    0,
			wantRealizedGain: ((75 * 170) - (75 * 150)) + ((100 * 190) - ((25 * 150) + (50 * 160) + (25 * 180))),
		},
		{
			name:             "Average Cost - After first sell",
			ticker:           "AAPL",
			date:             date.New(2025, time.February, 5),
			method:           AverageCost,
			wantCostBasis:    11500,
			wantRealizedGain: 1250,
		},
		{
			name:             "Average Cost - Final",
			ticker:           "AAPL",
			date:             date.New(2025, time.April, 1),
			method:           AverageCost,
			wantCostBasis:    0,
			wantRealizedGain: 4250,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			costBasis, err := ledger.CostBasis(tc.ticker, tc.date, tc.method)
			if err != nil {
				t.Errorf("CostBasis() error = %v", err)
			}
			if costBasis != tc.wantCostBasis {
				t.Errorf("CostBasis() = %v, want %v", costBasis, tc.wantCostBasis)
			}

			realizedGain, err := ledger.RealizedGain(tc.ticker, tc.date, tc.method)
			if err != nil {
				t.Errorf("RealizedGain() error = %v", err)
			}
			if realizedGain != tc.wantRealizedGain {
				t.Errorf("RealizedGain() = %v, want %v", realizedGain, tc.wantRealizedGain)
			}
		})
	}
}
