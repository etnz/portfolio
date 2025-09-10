package portfolio

import (
	"testing"
	"time"
)

// Validation of the accounting system needs to be done before computing.
// func TestAccountingSystem_CostBasis_ErrorOnMissingRate(t *testing.T) {
// 	ledger, _, _ := setupCostBasisTest(t)

// 	// Create market data that is missing a required exchange rate
// 	marketDataWithoutRate := NewMarketData()
// 	as, err := NewAccountingSystem(ledger, marketDataWithoutRate, "EUR")
// 	if err != nil {
// 		t.Fatalf("NewAccountingSystem() failed: %v", err)
// 	}

// 	_, err = as.CostBasis(date.New(2025, time.May, 1))
// 	if err == nil {
// 		t.Error("CostBasis() expected an error due to missing exchange rate, but got nil")
// 	}
// }

// setupValidationTest creates a standard ledger, market data, and accounting system for validation tests.
func setupValidationTest(t *testing.T) *AccountingSystem {
	t.Helper()

	o := NewDate(2020, time.January, 1)
	ledger := NewLedger()
	ledger.Append( //
		NewDeclare(o, "", "AAPL", AAPL, "USD"),
		NewDeclare(o, "", "GOOG", GOOG, "USD"),
		NewDeposit(NewDate(2025, time.January, 1), "", USD(20000), ""),
		NewDeposit(NewDate(2025, time.January, 1), "", EUR(10000), ""),
		NewBuy(NewDate(2025, time.January, 2), "", "AAPL", Q(100), USD(100*150.0)), // Cost: 15000 USD, remaining: 5000 USD
	)

	marketData := NewMarketData()
	aapl := NewSecurity(AAPL, "AAPL", "USD")
	goog := NewSecurity(GOOG, "GOOG", "USD")
	marketData.Add(aapl)
	marketData.Add(goog)

	// Reporting currency doesn't matter much for validation, but we set it for completeness.
	as, err := NewAccountingSystem(ledger, marketData, "EUR")
	if err != nil {
		t.Fatalf("NewAccountingSystem() failed: %v", err)
	}
	return as
}

func TestAccountingSystem_Validate(t *testing.T) {

	// Some test have been disable because the validation cannot be done with only the ledger.
	as := setupValidationTest(t)
	testDate := NewDate(2025, time.January, 10)

	testCases := []struct {
		name    string
		inputTx Transaction
		wantTx  Transaction
		wantErr bool
	}{
		// {
		// 	name: "Quick Fix: Sell All",
		// 	inputTx: NewSell(testDate, "sell all", "AAPL", Q(0), decimal.NewFromFloat(16000.0)),
		// 	wantTx:  NewSell(testDate, "sell all", "AAPL", Q(100), decimal.NewFromFloat(16000.0)), // Position is 100
		// 	wantErr: false,
		// },
		{
			name:    "Quick Fix: Withdraw All",
			inputTx: NewWithdraw(testDate, "cash out", USD(0)),
			wantTx:  NewWithdraw(testDate, "cash out", USD(5000)), // Balance is 5000 USD
			wantErr: false,
		},
		{
			name:    "Quick Fix: Convert All",
			inputTx: NewConvert(testDate, "fx", USD(0), EUR(4500)),
			wantTx:  NewConvert(testDate, "fx", USD(5000), EUR(4500)), // Balance is 5000
			wantErr: false,
		},
		{
			name:    "Quick Fix: Auto-populate date",
			inputTx: NewDeposit(Date{}, "late deposit", EUR(1000), ""),
			wantTx:  NewDeposit(Today(), "late deposit", EUR(1000), ""),
			wantErr: false,
		},
		{
			name:    "Error: Insufficient funds for Buy",
			inputTx: NewBuy(testDate, "", "AAPL", Q(1), USD(5001)), // Cost > 5000 balance
			wantErr: true,
		},
		{
			name:    "Error: Insufficient position for Sell",
			inputTx: NewSell(testDate, "", "AAPL", Q(101), USD(101*150)), // Position is 100
			wantErr: true,                                                // Position is 100
		},
		{
			name:    "Error: Invalid currency",
			inputTx: NewDeposit(testDate, "", M(1000, "US"), ""), // Invalid currency code,
			wantErr: true,
		},
		{
			name:    "Error: Negative quantity on Buy", // Quantity must be positive
			inputTx: NewBuy(testDate, "", "AAPL", Q(-10), USD(-10*150)),
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
				if !tc.wantTx.Equal(gotTx) { // Compare the actual structs, not just their interface values
					t.Errorf("Validate() got = %+v, want %+v", gotTx, tc.wantTx)
				}
			}
		})
	}
}
