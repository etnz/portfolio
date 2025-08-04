package portfolio

import (
	"reflect"
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
)

// setupCostBasisTest creates a standard ledger, market data, and accounting system for testing.
func setupCostBasisTest(t *testing.T) (*Ledger, *MarketData, *AccountingSystem) {
	t.Helper()

	// Create a ledger with deposits and withdrawals in different currencies.
	ledger := &Ledger{
		transactions: []Transaction{
			NewDeposit(date.New(2025, time.January, 10), "Initial USD", "USD", 1000),
			NewDeposit(date.New(2025, time.February, 15), "Initial GBP", "GBP", 500),
			NewWithdraw(date.New(2025, time.March, 20), "Partial USD", "USD", 200),
			NewDeposit(date.New(2025, time.April, 1), "EUR Deposit", "EUR", 2000),
			// Add a non-cash-flow transaction to ensure it's ignored
			NewBuy(date.New(2025, time.April, 5), "", "AAPL", 10, 150, "USD"),
		},
	}
	ledger.stableSort() // Ensure transactions are sorted by date

	// Create market data with historical exchange rates to EUR.
	marketData := NewMarketData()
	// USDEUR security for exchange rates
	usdeur := &Security{ticker: "USDEUR", id: "USDEUR", currency: "EUR"}
	usdeur.prices.Append(date.New(2025, time.January, 10), 0.90) // Rate on day of first USD deposit
	usdeur.prices.Append(date.New(2025, time.March, 20), 0.92)   // Rate on day of USD withdrawal
	marketData.securities = append(marketData.securities, usdeur)
	marketData.index["USDEUR"] = usdeur

	// GBPEUR security for exchange rates
	gbpeur := &Security{ticker: "GBPEUR", id: "GBPEUR", currency: "EUR"}
	gbpeur.prices.Append(date.New(2025, time.February, 15), 1.15) // Rate on day of GBP deposit
	marketData.securities = append(marketData.securities, gbpeur)
	marketData.index["GBPEUR"] = gbpeur

	// Create the accounting system with EUR as the reporting currency.
	as, err := NewAccountingSystem(ledger, marketData, "EUR")
	if err != nil {
		t.Fatalf("NewAccountingSystem() failed: %v", err)
	}
	return ledger, marketData, as
}

func TestAccountingSystem_CostBasis(t *testing.T) {
	_, _, as := setupCostBasisTest(t)

	testCases := []struct {
		name          string
		onDate        date.Date
		wantCostBasis float64
	}{
		{
			name:          "Before any transactions",
			onDate:        date.New(2025, time.January, 9),
			wantCostBasis: 0,
		},
		{
			name:          "On date of first deposit",
			onDate:        date.New(2025, time.January, 10),
			wantCostBasis: 900, // 1000 USD * 0.90
		},
		{
			name:          "After second deposit",
			onDate:        date.New(2025, time.February, 20),
			wantCostBasis: 1475, // (1000 * 0.90) + (500 * 1.15) = 900 + 575
		},
		{
			name:          "After withdrawal",
			onDate:        date.New(2025, time.March, 25),
			wantCostBasis: 1291, // 1475 - (200 * 0.92) = 1475 - 184
		},
		{
			name:          "After all cash flow transactions",
			onDate:        date.New(2025, time.May, 1),
			wantCostBasis: 3291, // 1291 + 2000 (EUR deposit)
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotCostBasis, err := as.CostBasis(tc.onDate)
			if err != nil {
				t.Fatalf("CostBasis() returned unexpected error: %v", err)
			}
			if gotCostBasis != tc.wantCostBasis {
				t.Errorf("CostBasis() = %v, want %v", gotCostBasis, tc.wantCostBasis)
			}
		})
	}
}

func TestAccountingSystem_CostBasis_ErrorOnMissingRate(t *testing.T) {
	ledger, _, _ := setupCostBasisTest(t)

	// Create market data that is missing a required exchange rate
	marketDataWithoutRate := NewMarketData()
	as, err := NewAccountingSystem(ledger, marketDataWithoutRate, "EUR")
	if err != nil {
		t.Fatalf("NewAccountingSystem() failed: %v", err)
	}

	_, err = as.CostBasis(date.New(2025, time.May, 1))
	if err == nil {
		t.Error("CostBasis() expected an error due to missing exchange rate, but got nil")
	}
}

// setupValidationTest creates a standard ledger, market data, and accounting system for validation tests.
func setupValidationTest(t *testing.T) *AccountingSystem {
	t.Helper()

	ledger := &Ledger{
		transactions: []Transaction{
			NewDeposit(date.New(2025, time.January, 1), "", "USD", 20000),
			NewDeposit(date.New(2025, time.January, 1), "", "EUR", 10000),
			NewBuy(date.New(2025, time.January, 2), "", "AAPL", 100, 150.0, "USD"), // Cost: 15000 USD, remaining: 5000 USD
		},
	}
	ledger.stableSort()

	marketData := NewMarketData()
	aapl := &Security{ticker: "AAPL", id: "US0378331005.XNAS", currency: "USD"}
	goog := &Security{ticker: "GOOG", id: "US38259P5089.XNAS", currency: "USD"}
	marketData.securities = append(marketData.securities, aapl, goog)
	marketData.index["AAPL"] = aapl
	marketData.index["GOOG"] = goog

	// Reporting currency doesn't matter much for validation, but we set it for completeness.
	as, err := NewAccountingSystem(ledger, marketData, "EUR")
	if err != nil {
		t.Fatalf("NewAccountingSystem() failed: %v", err)
	}
	return as
}

func TestAccountingSystem_Validate(t *testing.T) {
	as := setupValidationTest(t)
	testDate := date.New(2025, time.January, 10)

	testCases := []struct {
		name    string
		inputTx Transaction
		wantTx  Transaction
		wantErr bool
	}{
		{
			name:    "Quick Fix: Sell All",
			inputTx: NewSell(testDate, "sell all", "AAPL", 0, 160.0, "USD"),
			wantTx:  NewSell(testDate, "sell all", "AAPL", 100, 160.0, "USD"), // Position is 100
			wantErr: false,
		},
		{
			name:    "Quick Fix: Withdraw All",
			inputTx: NewWithdraw(testDate, "cash out", "USD", 0),
			wantTx:  NewWithdraw(testDate, "cash out", "USD", 5000), // Balance is 5000
			wantErr: false,
		},
		{
			name:    "Quick Fix: Convert All",
			inputTx: NewConvert(testDate, "fx", "USD", 0, "EUR", 4500),
			wantTx:  NewConvert(testDate, "fx", "USD", 5000, "EUR", 4500), // Balance is 5000
			wantErr: false,
		},
		{
			name:    "Quick Fix: Auto-populate currency",
			inputTx: NewBuy(testDate, "", "GOOG", 10, 280, ""), // Empty currency
			wantTx:  NewBuy(testDate, "", "GOOG", 10, 280, "USD"),
			wantErr: false,
		},
		{
			name:    "Quick Fix: Auto-populate date",
			inputTx: NewDeposit(date.Date{}, "late deposit", "EUR", 1000), // Zero date
			wantTx:  NewDeposit(date.Today(), "late deposit", "EUR", 1000),
			wantErr: false,
		},
		{
			name:    "Error: Insufficient funds for Buy",
			inputTx: NewBuy(testDate, "", "AAPL", 1, 5001, "USD"), // Cost > 5000 balance
			wantErr: true,
		},
		{
			name:    "Error: Insufficient position for Sell",
			inputTx: NewSell(testDate, "", "AAPL", 101, 150, "USD"), // Position is 100
			wantErr: true,
		},
		{
			name:    "Error: Invalid currency",
			inputTx: NewDeposit(testDate, "", "US", 1000), // Invalid currency code
			wantErr: true,
		},
		{
			name:    "Error: Negative quantity on Buy",
			inputTx: NewBuy(testDate, "", "AAPL", -10, 150, "USD"),
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotTx, err := as.Validate(tc.inputTx)

			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !tc.wantErr {
				if !reflect.DeepEqual(gotTx, tc.wantTx) {
					t.Errorf("Validate() got = %+v, want %+v", gotTx, tc.wantTx)
				}
			}
		})
	}
}
