package portfolio

import (
	"testing"
	"time"
)

func TestAccountingSystem_CalculateGains_FilterZeroGainRows(t *testing.T) {
	ledger := NewLedger()
	ledger.currency = "USD"
	o := NewDate(2025, time.January, 1)
	ZEROID, _ := NewPrivate("ZERO-ID")
	ledger.Append(
		NewDeclare(o, "", "ZERO", ZEROID, "USD"),
		NewBuy(NewDate(2025, time.January, 10), "", "ZERO", Q(100), USD(100*100)), // Cost: 10000
		// Add a price for ZERO
		NewUpdatePrice(NewDate(2025, time.February, 1), "ZERO", USD(100.0)),
	)

	report, err := NewGainsReport(ledger, Range{From: NewDate(2025, time.March, 1), To: NewDate(2025, time.March, 31)}, AverageCost)
	if err != nil {
		t.Fatalf("NewGainsReport() error = %v", err)
	}

	// Period where ZERO has no activity and no change in price
	if len(report.Securities) != 0 {
		t.Errorf("Expected 0 securities in report, got %d: %v", len(report.Securities), report.Securities)
	}
}
