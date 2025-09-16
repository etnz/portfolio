package portfolio

import (
	"testing"
)

func TestReview_PeriodWithInvestmentAndGains(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "EUR"

	txs := []Transaction{
		// --- BEFORE Period ---
		NewDeclare(NewDate(2025, 1, 1), "", "AAPL", AAPL, "EUR"),
		NewDeposit(NewDate(2025, 1, 2), "", EUR(10000), ""),
		NewBuy(NewDate(2025, 1, 3), "", "AAPL", Q(10), EUR(1000)), // Price: 100
		NewUpdatePrice(NewDate(2025, 1, 4), "AAPL", EUR(110)),

		// --- DURING Period ---
		// Day 1: Deposit and Price change
		NewDeposit(NewDate(2025, 1, 5), "", EUR(5000), ""),
		NewUpdatePrice(NewDate(2025, 1, 5), "AAPL", EUR(120)),
		// Day 2: Dividend
		NewDividend(NewDate(2025, 1, 6), "", "AAPL", EUR(5)), // 5 EUR/share * 10 shares = 50 EUR
		// Day 3: Sell
		NewSell(NewDate(2025, 1, 7), "", "AAPL", Q(5), EUR(650)), // Sell 5 shares @ 130
		NewUpdatePrice(NewDate(2025, 1, 7), "AAPL", EUR(130)),
	}
	if err := ledger.Append(txs...); err != nil {
		t.Fatalf("ledger.Append() error = %v", err)
	}

	period := NewRange(NewDate(2025, 1, 5), NewDate(2025, 1, 7))
	review, err := ledger.NewReview(period)
	if err != nil {
		t.Fatalf("NewReview() error = %v", err)
	}

	// --- Verification ---

	// startSnapshot is on 2025-01-04
	// endSnapshot is on 2025-01-07

	// Base Metrics
	t.Run("Base Metrics", func(t *testing.T) {
		// CashFlow: Only the 5000 EUR deposit during the period is external cash flow.
		if got, want := review.CashFlow(), EUR(5000); !got.Equal(want) {
			t.Errorf("CashFlow() = %v, want %v", got, want)
		}

		// Dividends: 50 EUR dividend was received during the period.
		if got, want := review.Dividends(), EUR(50); !got.Equal(want) {
			t.Errorf("Dividends() = %v, want %v", got, want)
		}

		// RealizedGains (FIFO):
		// 5 shares sold for 650. Cost of first 5 shares was 5 * 100 = 500.
		// Realized gain = 650 - 500 = 150.
		if got, want := review.RealizedGains(FIFO), EUR(150); !got.Equal(want) {
			t.Errorf("RealizedGains(FIFO) = %v, want %v", got, want)
		}

		// TimeWeightedReturn:
		// VAV start (Jan 4): 1.1 (from 10% gain on Jan 3-4)
		// VAV end (Jan 7):
		// Jan 5: Price 110 -> 120. Factor = 120/110. VAV = 1.1 * (120/110) = 1.2
		// Jan 6: Dividend. VAV is unaffected.
		// Jan 7: Price 120 -> 130. Factor = 130/120. VAV = 1.2 * (130/120) = 1.3
		// Total return = (1.3 / 1.1) - 1 = 0.1818... or 18.18%
		if got, want := review.TimeWeightedReturn("AAPL"), Percent(18.181818); !got.Equal(want) {
			t.Errorf("TimeWeightedReturn() = %v, want %v", got, want)
		}
	})

	// Compound Metrics
	t.Run("Compound Metrics", func(t *testing.T) {
		// MarketGainLoss = (TotalMarketValue(end) - TotalMarketValue(start)) - NetTradingFlow
		// start TMV (Jan 4): 10 shares * 110 = 1100
		// end TMV (Jan 7): 5 shares * 130 = 650
		// TMV Change = 650 - 1100 = -450.
		// NetTradingFlow is Buys - Sells. During the period, Buys = 0, Sells = 650. So NetTradingFlow = -650.
		// MarketGainLoss = -450 - (-650) = 200.
		//
		// Manual verification:
		// Gain from Jan 4 to Jan 5: 10 shares * (120-110) = +100
		// Gain from Jan 5 to Jan 7 on 5 held shares: 5 * (130-120) = +50
		// Gain from Jan 5 to Jan 7 on 5 sold shares: 5 * (130-120) = +50
		// Total Market Gain = 100 + 50 + 50 = 200. The formula is correct.

		// Correcting the NetTradingFlow test first.
		if got, want := review.NetTradingFlow(), EUR(-650); !got.Equal(want) {
			t.Errorf("NetTradingFlow() = %v, want %v", got, want) // Buys are positive flow, Sells are negative.
		}

		if got, want := review.MarketGainLoss(), EUR(200); !got.Equal(want) {
			t.Errorf("MarketGainLoss() = %v, want %v", got, want)
		}

		// TotalReturn = MarketGainLoss + Dividends
		// 200 (Market Gain) + 50 (Dividends) = 250
		if got, want := review.TotalReturn(), EUR(250); !got.Equal(want) {
			t.Errorf("TotalReturn() = %v, want %v", got, want)
		}
	})
}
