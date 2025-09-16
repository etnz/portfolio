package portfolio

import (
	"math"
	"testing"
)

func TestSnapshot_EmptyPortfolio(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "EUR"
	if err := ledger.Append(); err != nil {
		t.Fatalf("ledger.Append() error = %v", err)
	}

	date := NewDate(2025, 1, 1)
	s, err := ledger.NewSnapshot(date)
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}

	t.Run("Point-in-time metrics are zero", func(t *testing.T) {
		if !s.Position("ANY").IsZero() {
			t.Error("Position should be zero for an empty portfolio")
		}
		if !s.Cash("ANY").IsZero() {
			t.Error("Cash should be zero for an empty portfolio")
		}
		if !s.Counterparty("ANY").IsZero() {
			t.Error("Counterparty should be zero for an empty portfolio")
		}
		if !s.Price("ANY").IsZero() {
			t.Error("Price should be zero for an empty portfolio")
		}
		if !s.MarketValue("ANY").IsZero() {
			t.Error("MarketValue should be zero for an empty portfolio")
		}
		if !s.CostBasis("ANY", AverageCost).IsZero() {
			t.Error("CostBasis should be zero for an empty portfolio")
		}
		if !s.UnrealizedGains("ANY", AverageCost).IsZero() {
			t.Error("UnrealizedGains should be zero for an empty portfolio")
		}
	})

	t.Run("Cumulative metrics are zero", func(t *testing.T) {
		if !s.RealizedGains("ANY", AverageCost).IsZero() {
			t.Error("RealizedGains should be zero for an empty portfolio")
		}
		if !s.Dividends("ANY").IsZero() {
			t.Error("Dividends should be zero for an empty portfolio")
		}
		if !s.NetTradingFlow("ANY").IsZero() {
			t.Error("NetTradingFlow should be zero for an empty portfolio")
		}
		if !s.CashFlow("ANY").IsZero() {
			t.Error("CashFlow should be zero for an empty portfolio")
		}
	})

	t.Run("Total portfolio metrics are zero", func(t *testing.T) {
		if !s.TotalMarket().IsZero() {
			t.Error("TotalMarket should be zero for an empty portfolio")
		}
		if !s.TotalCash().IsZero() {
			t.Error("TotalCash should be zero for an empty portfolio")
		}
		if !s.TotalCounterparty().IsZero() {
			t.Error("TotalCounterparty should be zero for an empty portfolio")
		}
		if !s.TotalPortfolio().IsZero() {
			t.Error("TotalPortfolio should be zero for an empty portfolio")
		}
	})

	t.Run("VirtualAssetValue starts at 1", func(t *testing.T) {
		// The virtual asset value of a 1-unit investment that is never made is still 1.
		want := EUR(1)
		got := s.VirtualAssetValue("ANY")
		if !got.Equal(want) {
			t.Errorf("VirtualAssetValue() = %v, want %v", got, want)
		}
	})
}

func TestSnapshot_BasicSingleSecurity(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "EUR"

	txs := []Transaction{
		NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
		NewDeposit(NewDate(2025, 1, 2), "", EUR(10000), ""),
		NewBuy(NewDate(2025, 1, 3), "", "AAPL", Q(10), EUR(1500)),
		NewUpdatePrice(NewDate(2025, 1, 4), "AAPL", EUR(160)),
	}
	if err := ledger.Append(txs...); err != nil {
		t.Fatalf("ledger.Append() error = %v", err)
	}

	s, err := ledger.NewSnapshot(NewDate(2025, 1, 4))
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}

	if got, want := s.Position("AAPL"), Q(10); !got.Equal(want) {
		t.Errorf("Position() = %v, want %v", got, want)
	}
	if got, want := s.Cash("EUR"), EUR(8500); !got.Equal(want) {
		t.Errorf("Cash() = %v, want %v", got, want)
	}
	if got, want := s.CostBasis("AAPL", AverageCost), EUR(1500); !got.Equal(want) {
		t.Errorf("CostBasis() = %v, want %v", got, want)
	}
	if got, want := s.Price("AAPL"), EUR(160); !got.Equal(want) {
		t.Errorf("Price() = %v, want %v", got, want)
	}
	if got, want := s.MarketValue("AAPL"), EUR(1600); !got.Equal(want) {
		t.Errorf("MarketValue() = %v, want %v", got, want)
	}
	if got, want := s.UnrealizedGains("AAPL", AverageCost), EUR(100); !got.Equal(want) {
		t.Errorf("UnrealizedGains() = %v, want %v", got, want)
	}
	if got, want := s.TotalMarket(), EUR(1600); !got.Equal(want) {
		t.Errorf("TotalMarket() = %v, want %v", got, want)
	}
	if got, want := s.TotalCash(), EUR(8500); !got.Equal(want) {
		t.Errorf("TotalCash() = %v, want %v", got, want)
	}
	if got, want := s.TotalPortfolio(), EUR(10100); !got.Equal(want) {
		t.Errorf("TotalPortfolio() = %v, want %v", got, want)
	}
}

func TestSnapshot_MultiCurrency(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "EUR"
	txs := []Transaction{
		NewDeclare(NewDate(2025, 1, 1), "", "MSFT", "US5949181045.XNAS", "USD"), // Not using a constant for MSFT as it's not defined in helpers
		NewDeclare(NewDate(2025, 1, 1), "", "USDEUR", USDEUR, "EUR"),
		NewDeposit(NewDate(2025, 1, 2), "", EUR(10000), ""),
		NewDeposit(NewDate(2025, 1, 2), "", USD(5000), ""),
		NewUpdatePrice(NewDate(2025, 1, 2), "USDEUR", EUR(0.9)),
		NewBuy(NewDate(2025, 1, 3), "", "MSFT", Q(10), USD(4000)),
		NewUpdatePrice(NewDate(2025, 1, 4), "MSFT", USD(420)),
		NewUpdatePrice(NewDate(2025, 1, 4), "USDEUR", EUR(0.92)),
	}
	if err := ledger.Append(txs...); err != nil {
		t.Fatalf("ledger.Append() error = %v", err)
	}

	s, err := ledger.NewSnapshot(NewDate(2025, 1, 4))
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}

	// Per-currency metrics
	if got, want := s.Cash("EUR"), EUR(10000); !got.Equal(want) {
		t.Errorf("Cash(EUR) = %v, want %v", got, want)
	}
	if got, want := s.Cash("USD"), USD(1000); !got.Equal(want) {
		t.Errorf("Cash(USD) = %v, want %v", got, want)
	}
	if got, want := s.MarketValue("MSFT"), USD(4200); !got.Equal(want) {
		t.Errorf("MarketValue(MSFT) = %v, want %v", got, want)
	}
	if got, want := s.ExchangeRate("USD"), EUR(0.92); !got.Equal(want) {
		t.Errorf("ExchangeRate(USD) = %v, want %v", got, want)
	}

	// Total portfolio metrics (converted to EUR)
	expectedTotalCash := EUR(10000).Add(s.Convert(USD(1000))) // 10000 + 1000*0.92 = 10920
	if got := s.TotalCash(); !got.Equal(expectedTotalCash) {
		t.Errorf("TotalCash() = %v, want %v", got, expectedTotalCash)
	}

	expectedTotalMarket := s.Convert(M(4200, "USD")) // 4200 * 0.92 = 3864
	if got := s.TotalMarket(); !got.Equal(expectedTotalMarket) {
		t.Errorf("TotalMarket() = %v, want %v", got, expectedTotalMarket)
	}

	expectedTotalPortfolio := expectedTotalCash.Add(expectedTotalMarket) // 10920 + 3864 = 14784
	if got := s.TotalPortfolio(); !got.Equal(expectedTotalPortfolio) {
		t.Errorf("TotalPortfolio() = %v, want %v", got, expectedTotalPortfolio)
	}
}

func TestSnapshot_RealizedGains(t *testing.T) {
	tests := []struct {
		name               string
		method             CostBasisMethod
		expectedCostBasis  Money
		expectedRealized   Money
		expectedUnrealized Money
	}{
		{
			name:               "AverageCost",
			method:             AverageCost,
			expectedCostBasis:  EUR(1100), // (1000+1200)/20 * 10
			expectedRealized:   EUR(150),  // 1250 - (1000+1200)/20 * 10
			expectedUnrealized: EUR(200),  // 1300 - 1100
		},
		{
			name:               "FIFO",
			method:             FIFO,
			expectedCostBasis:  EUR(1200), // Second lot remains
			expectedRealized:   EUR(250),  // 1250 - 1000 (cost of first lot)
			expectedUnrealized: EUR(100),  // 1300 - 1200
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := NewLedger()
			ledger.currency = "EUR"
			txs := []Transaction{
				NewDeclare(NewDate(2025, 1, 1), "", "AAPL", "US0378331005.XNAS", "EUR"),
				NewDeposit(NewDate(2025, 1, 2), "", M(10000, "EUR"), ""),
				NewBuy(NewDate(2025, 1, 3), "", "AAPL", Q(10), M(1000, "EUR")), // Lot 1: 10 @ 100
				NewBuy(NewDate(2025, 1, 4), "", "AAPL", Q(10), M(1200, "EUR")), // Lot 2: 10 @ 120
				NewSell(NewDate(2025, 1, 5), "", "AAPL", Q(10), M(1250, "EUR")),
				NewUpdatePrice(NewDate(2025, 1, 5), "AAPL", M(130, "EUR")),
			}
			if err := ledger.Append(txs...); err != nil {
				t.Fatalf("ledger.Append() error = %v", err)
			}

			s, err := ledger.NewSnapshot(NewDate(2025, 1, 5))
			if err != nil {
				t.Fatalf("NewSnapshot() error = %v", err)
			}

			if got, want := s.Position("AAPL"), Q(10); !got.Equal(want) {
				t.Errorf("Position() = %v, want %v", got, want)
			}
			if got := s.CostBasis("AAPL", tt.method); !got.Equal(tt.expectedCostBasis) {
				t.Errorf("CostBasis() = %v, want %v", got, tt.expectedCostBasis)
			}
			if got := s.RealizedGains("AAPL", tt.method); !got.Equal(tt.expectedRealized) {
				t.Errorf("RealizedGains() = %v, want %v", got, tt.expectedRealized)
			}
			if got := s.UnrealizedGains("AAPL", tt.method); !got.Equal(tt.expectedUnrealized) {
				t.Errorf("UnrealizedGains() = %v, want %v", got, tt.expectedUnrealized)
			}
		})
	}
}

func TestSnapshot_CorporateActions(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "EUR"
	txs := []Transaction{
		NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
		NewDeposit(NewDate(2025, 1, 2), "", EUR(10000), ""),
		NewBuy(NewDate(2025, 1, 3), "", "AAPL", Q(10), EUR(1500)),
		NewDividend(NewDate(2025, 1, 4), "", "AAPL", EUR(5)),
		NewSplit(NewDate(2025, 1, 5), "AAPL", 2, 1),
	}
	if err := ledger.Append(txs...); err != nil {
		t.Fatalf("ledger.Append() error = %v", err)
	}

	s, err := ledger.NewSnapshot(NewDate(2025, 1, 5))
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}

	t.Run("Dividends", func(t *testing.T) {
		// Dividend is per-share, so 10 shares * 5 EUR/share = 50 EUR
		if got, want := s.Dividends("AAPL"), EUR(50); !got.Equal(want) {
			t.Errorf("Dividends() = %v, want %v", got, want)
		}
		// Cash balance should NOT reflect the dividend received, as it's external income.
		if got, want := s.Cash("EUR"), EUR(8500); !got.Equal(want) { // 10000 - 1500
			t.Errorf("Cash() = %v, want %v", got, want)
		}
	})

	t.Run("Split", func(t *testing.T) {
		// Position doubles from 10 to 20
		if got, want := s.Position("AAPL"), Q(20); !got.Equal(want) {
			t.Errorf("Position() = %v, want %v", got, want)
		}
		// Cost basis remains the same, spread over more shares
		if got, want := s.CostBasis("AAPL", AverageCost), EUR(1500); !got.Equal(want) {
			t.Errorf("CostBasis() = %v, want %v", got, want)
		}
	})
}

func TestSnapshot_CounterpartyAccounts(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "EUR"
	txs := []Transaction{
		NewCreatedAccrue(NewDate(2025, 1, 1), "", "TAXMAN", EUR(0)),
		NewCreatedAccrue(NewDate(2025, 1, 1), "", "CLIENT", EUR(0)),
		NewDeposit(NewDate(2025, 1, 2), "", EUR(10000), ""),
		// Accrue a liability (money owed)
		NewAccrue(NewDate(2025, 1, 3), "", "TAXMAN", EUR(-500)),
		// Accrue a receivable (money to be received)
		NewAccrue(NewDate(2025, 1, 4), "", "CLIENT", EUR(1000)),
		// Settle the receivable: client pays us. This is NOT an external cash flow.
		NewDeposit(NewDate(2025, 1, 5), "", EUR(1000), "CLIENT"),
	}
	if err := ledger.Append(txs...); err != nil {
		t.Fatalf("ledger.Append() error = %v", err)
	}

	s, err := ledger.NewSnapshot(NewDate(2025, 1, 5))
	if err != nil {
		t.Fatalf("NewSnapshot() error = %v", err)
	}

	t.Run("Counterparty Balances", func(t *testing.T) {
		// Liability: we owe 500
		if got, want := s.Counterparty("TAXMAN"), EUR(-500); !got.Equal(want) {
			t.Errorf("Counterparty(TAXMAN) = %v, want %v", got, want)
		}
		// Receivable was accrued, then settled. Balance should be zero.
		if got, want := s.Counterparty("CLIENT"), EUR(0); !got.Equal(want) {
			t.Errorf("Counterparty(CLIENT) = %v, want %v", got, want)
		}
		// Total is just the outstanding liability
		if got, want := s.TotalCounterparty(), EUR(-500); !got.Equal(want) {
			t.Errorf("TotalCounterparty() = %v, want %v", got, want)
		}
	})

	t.Run("Cash Balance and Flow", func(t *testing.T) {
		// 10000 (initial) + 1000 (from client) = 11000
		if got, want := s.Cash("EUR"), EUR(11000); !got.Equal(want) {
			t.Errorf("Cash() = %v, want %v", got, want)
		}
		// The deposit from the client was internal (settling a receivable),
		// so the only external cash flow is the initial 10000 deposit.
		if got, want := s.CashFlow("EUR"), EUR(10000); !got.Equal(want) {
			t.Errorf("CashFlow() = %v, want %v", got, want)
		}
	})

	t.Run("Total Portfolio Value", func(t *testing.T) {
		// Cash (11000) + Counterparties (-500) = 10500
		if got, want := s.TotalPortfolio(), EUR(10500); !got.Equal(want) {
			t.Errorf("TotalPortfolio() = %v, want %v", got, want)
		}
	})
}

func TestSnapshot_TimeWeightedReturn_VirtualAssetValue(t *testing.T) {
	t.Run("Simple buy and hold with price increase", func(t *testing.T) {
		ledger := NewLedger()
		ledger.currency = "EUR"
		txs := []Transaction{
			NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
			NewBuy(NewDate(2025, 1, 2), "", "AAPL", Q(10), EUR(1000)), // Price is 100
			NewUpdatePrice(NewDate(2025, 1, 3), "AAPL", EUR(110)),
		}
		ledger.Append(txs...)
		s, _ := ledger.NewSnapshot(NewDate(2025, 1, 3))

		// Virtual portfolio buys 1 EUR worth of stock at 100 EUR/share, gets 0.01 shares.
		// Value at end is 0.01 shares * 110 EUR/share = 1.1 EUR.
		// This represents a 10% return.
		if got, want := s.VirtualAssetValue("AAPL"), EUR(1.1); !got.Equal(want) {
			t.Errorf("VirtualAssetValue() = %v, want %v", got, want)
		}
	})

	t.Run("Price increase with intermediate cash flow", func(t *testing.T) {
		ledger := NewLedger()
		ledger.currency = "EUR"
		txs := []Transaction{
			NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
			NewBuy(NewDate(2025, 1, 2), "", "AAPL", Q(10), EUR(1000)), // Price is 100
			NewUpdatePrice(NewDate(2025, 1, 3), "AAPL", EUR(110)),     // Up 10%
			NewDeposit(NewDate(2025, 1, 3), "", EUR(5000), ""),        // <-- Cash flow, should not affect TWR
			NewUpdatePrice(NewDate(2025, 1, 4), "AAPL", EUR(121)),     // Up another 10%
		}
		ledger.Append(txs...)
		s, _ := ledger.NewSnapshot(NewDate(2025, 1, 4))

		// Virtual portfolio buys 1 EUR of stock -> 0.01 shares.
		// Value becomes 0.01 * 110 = 1.1 EUR.
		// Value becomes 0.01 * 121 = 1.21 EUR.
		// This represents a 21% return (1.1 * 1.1).
		if got, want := s.VirtualAssetValue("AAPL"), EUR(1.21); !got.Equal(want) {
			t.Errorf("VirtualAssetValue() = %v, want %v", got, want)
		}
	})

	t.Run("Sell out and buy back in", func(t *testing.T) {
		ledger := NewLedger()
		ledger.currency = "EUR"
		txs := []Transaction{
			NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
			NewBuy(NewDate(2025, 1, 2), "", "AAPL", Q(10), EUR(1000)),  // Price 100
			NewUpdatePrice(NewDate(2025, 1, 3), "AAPL", EUR(110)),      // Up 10%
			NewSell(NewDate(2025, 1, 4), "", "AAPL", Q(10), EUR(1100)), // Sell all
			NewBuy(NewDate(2025, 1, 5), "", "AAPL", Q(5), EUR(600)),    // Buy back in at price 120
			NewUpdatePrice(NewDate(2025, 1, 6), "AAPL", EUR(132)),      // Up 10% again
		}
		ledger.Append(txs...)
		s, _ := ledger.NewSnapshot(NewDate(2025, 1, 6))

		// Virtual portfolio:
		// 1. Buys 1 EUR of stock -> 0.01 shares @ 100.
		// 2. Value becomes 1.1 EUR @ 110.
		// 3. Sells all, virtual cash is now 1.1 EUR.
		// 4. Buys back in with 1.1 EUR of stock @ 120 -> 1.1/120 = 0.009166... shares.
		// 5. Value becomes (1.1/120) * 132 = 1.1 * 1.1 = 1.21 EUR.
		// Total return is 21%.
		vav := s.VirtualAssetValue("AAPL")
		if got, want, delta := vav.AsFloat(), 1.21, 0.00001; math.Abs(got-want) > delta {
			t.Errorf("VirtualAssetValue() = %v, want %v (within %v)", got, want, delta)
		}
	})

	t.Run("No investment made", func(t *testing.T) {
		ledger := NewLedger()
		ledger.currency = "EUR"
		txs := []Transaction{
			NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
			NewUpdatePrice(NewDate(2025, 1, 3), "AAPL", EUR(110)),
		}
		ledger.Append(txs...)
		s, _ := ledger.NewSnapshot(NewDate(2025, 1, 3))

		// Virtual portfolio starts with 1 EUR and never invests it.
		if got, want := s.VirtualAssetValue("AAPL"), EUR(1); !got.Equal(want) {
			t.Errorf("VirtualAssetValue() = %v, want %v", got, want)
		}
	})

	t.Run("Investment in another security", func(t *testing.T) {
		ledger := NewLedger()
		ledger.currency = "EUR"
		txs := []Transaction{
			NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
			NewDeclare(NewDate(2025, 1, 1), "", "MSFT", "US5949181045.XNAS", "EUR"),
			NewBuy(NewDate(2025, 1, 2), "", "MSFT", Q(10), EUR(1000)),
			NewUpdatePrice(NewDate(2025, 1, 3), "MSFT", EUR(110)),
		}
		ledger.Append(txs...)
		s, _ := ledger.NewSnapshot(NewDate(2025, 1, 3))

		// Virtual portfolio for AAPL starts with 1 EUR and never invests it.
		if got, want := s.VirtualAssetValue("AAPL"), EUR(1); !got.Equal(want) {
			t.Errorf("VirtualAssetValue(AAPL) = %v, want %v", got, want)
		}
		// Virtual portfolio for MSFT should show the 10% gain.
		if got, want := s.VirtualAssetValue("MSFT"), EUR(1.1); !got.Equal(want) {
			t.Errorf("VirtualAssetValue(MSFT) = %v, want %v", got, want)
		}
	})
}
