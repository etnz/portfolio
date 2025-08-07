package portfolio

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
)

// setupCostBasisTest creates a standard ledger, market data, and accounting system for testing.
func setupCostBasisTest(t *testing.T) (*Ledger, *MarketData, *AccountingSystem) {
	t.Helper()

	// Create a ledger with deposits and withdrawals in different currencies.
	o := date.New(2025, time.January, 1)

	ledger := NewLedger()
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "USD"),
		NewDeclaration(o, "", "USDEUR", "USDEUR", "EUR"),
		NewDeclaration(o, "", "GBPEUR", "GBPEUR", "EUR"),
		NewDeposit(date.New(2025, time.January, 10), "Initial USD", "USD", 1000),
		NewDeposit(date.New(2025, time.February, 15), "Initial GBP", "GBP", 500),
		NewWithdraw(date.New(2025, time.March, 20), "Partial USD", "USD", 200),
		NewDeposit(date.New(2025, time.April, 1), "EUR Deposit", "EUR", 2000),
		// Add a non-cash-flow transaction to ensure it's ignored
		NewBuy(date.New(2025, time.April, 5), "", "AAPL", 10, 150),
	)

	// Create market data with historical exchange rates to EUR.
	marketData := NewMarketData()
	// USDEUR security for exchange rates
	usdeur := &Security{ticker: "USDEUR", id: "USDEUR", currency: "EUR"}
	marketData.Add(usdeur)
	marketData.Append("USDEUR", date.New(2025, time.January, 10), 0.90) // Rate on day of first USD deposit
	marketData.Append("USDEUR", date.New(2025, time.March, 20), 0.92)   // Rate on day of USD withdrawal

	// GBPEUR security for exchange rates
	gbpeur := &Security{ticker: "GBPEUR", id: "GBPEUR", currency: "EUR"}
	marketData.Add(gbpeur)
	marketData.Append("GBPEUR", date.New(2025, time.February, 15), 1.15) // Rate on day of GBP deposit

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

// setupPerformanceTest creates a ledger and market data for performance calculation tests.
func setupPerformanceTest(t *testing.T) (*Ledger, *MarketData, *AccountingSystem) {
	t.Helper()

	id, err := NewPrivate("TICK Private Equity")
	if err != nil {
		t.Fatalf("NewPrivate() failed: %v", err)
	}

	o := date.New(2025, time.January, 1)

	ledger := NewLedger()
	ledger.Append(
		NewDeclaration(o, "", "TICK", id.String(), "USD"),
		NewDeposit(date.New(2025, time.January, 1), "", "USD", 10000),
		NewBuy(date.New(2025, time.January, 1), "", "TICK", 100, 100),
		NewDeposit(date.New(2025, time.January, 15), "", "USD", 1100),
	)

	marketData := NewMarketData()

	tick := &Security{ticker: "TICK", id: id, currency: "USD"}
	marketData.Add(tick)
	marketData.Append(id, date.New(2025, time.January, 1), 100.0)
	marketData.Append(id, date.New(2025, time.January, 15), 110.0)
	marketData.Append(id, date.New(2025, time.January, 31), 120.0)

	as, err := NewAccountingSystem(ledger, marketData, "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() failed: %v", err)
	}
	return ledger, marketData, as
}

func TestAccountingSystem_CalculatePeriodPerformance(t *testing.T) {
	_, _, as := setupPerformanceTest(t)

	startDate := date.New(2025, time.January, 1)
	endDate := date.New(2025, time.January, 31)

	perf, err := as.calculatePeriodPerformance(startDate, endDate)
	if err != nil {
		t.Fatalf("calculatePeriodPerformance() returned unexpected error: %v", err)
	}

	// Expected start value is TMV on Dec 31, 2024, which is 0.
	if perf.StartValue != 0 {
		t.Errorf("perf.StartValue = %v, want %v", perf.StartValue, 0)
	}

	// Expected return calculation:
	// Start Value (V0) = 0
	// Day 1 (Jan 1): Deposit 10k, Buy 100 TICK @ 100.
	//   V1_after_cf = 100 * 100 = 10000. CF1 = 10000. V1_before_cf = 0.
	//   Since V0 is 0, first period return is not calculated. V_last = 10000.
	// Day 15 (Jan 15): Deposit 1.1k. TICK price is 110.
	//   V2_after_cf = (100 * 110) + 1100 = 12100. CF2 = 1100. V2_before_cf = 11000.
	//   HPR2 = (11000 - 10000) / 10000 = 0.1.
	//   LinkedReturn = 1 * (1 + 0.1) = 1.1. V_last = 12100.
	// Day 31 (Jan 31): End of period. TICK price is 120.
	//   V3 = (100 * 120) + 1100 = 13100. No CF.
	//   HPR3 = (13100 - 12100) / 12100 = 1000 / 12100 = 0.0826446...
	//   LinkedReturn = 1.1 * (1 + 1000/12100) = 1.1 * (13100/12100) = 1.190909...
	// TWR = 1.190909... - 1 = 0.190909...
	const wantReturn = (1.1 * (13100.0 / 12100.0)) - 1.0
	const tolerance = 1e-9

	if math.Abs(perf.Return-wantReturn) > tolerance {
		t.Errorf("perf.Return = %v, want %v", perf.Return, wantReturn)
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

	o := date.New(2020, time.January, 1)
	ledger := NewLedger()
	ledger.Append(
		NewDeclaration(o, "", "AAPL", "US0378331005.XNAS", "USD"),
		NewDeclaration(o, "", "GOOG", "US38259P5089.XNAS", "USD"),
		NewDeposit(date.New(2025, time.January, 1), "", "USD", 20000),
		NewDeposit(date.New(2025, time.January, 1), "", "EUR", 10000),
		NewBuy(date.New(2025, time.January, 2), "", "AAPL", 100, 150.0), // Cost: 15000 USD, remaining: 5000 USD
	)

	marketData := NewMarketData()
	aapl := &Security{ticker: "AAPL", id: "US0378331005.XNAS", currency: "USD"}
	goog := &Security{ticker: "GOOG", id: "US38259P5089.XNAS", currency: "USD"}
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
			inputTx: NewSell(testDate, "sell all", "AAPL", 0, 160.0),
			wantTx:  NewSell(testDate, "sell all", "AAPL", 100, 160.0), // Position is 100
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
			name:    "Quick Fix: Auto-populate date",
			inputTx: NewDeposit(date.Date{}, "late deposit", "EUR", 1000), // Zero date
			wantTx:  NewDeposit(date.Today(), "late deposit", "EUR", 1000),
			wantErr: false,
		},
		{
			name:    "Error: Insufficient funds for Buy",
			inputTx: NewBuy(testDate, "", "AAPL", 1, 5001), // Cost > 5000 balance
			wantErr: true,
		},
		{
			name:    "Error: Insufficient position for Sell",
			inputTx: NewSell(testDate, "", "AAPL", 101, 150), // Position is 100
			wantErr: true,
		},
		{
			name:    "Error: Invalid currency",
			inputTx: NewDeposit(testDate, "", "US", 1000), // Invalid currency code
			wantErr: true,
		},
		{
			name:    "Error: Negative quantity on Buy",
			inputTx: NewBuy(testDate, "", "AAPL", -10, 150),
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
