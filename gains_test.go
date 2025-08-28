package portfolio

import (
	"testing"
	"time"

	"github.com/etnz/portfolio/date"
)

func TestAccountingSystem_CalculateGains_FilterZeroGainRows(t *testing.T) {
	ledger := NewLedger()
	o := date.New(2025, time.January, 1)
	ledger.Append(
		NewDeclaration(o, "", "ZERO", "ZERO-ID", "USD"),
		NewBuyWithPrice(date.New(2025, time.January, 10), "", "ZERO", 100, 100.0), // Cost: 10000
	)

	market := NewMarketData()
	market.Add(NewSecurity(must(NewPrivate("ZERO-ID")), "ZERO", "USD"))
	market.Append(must(NewPrivate("ZERO-ID")), date.New(2025, time.February, 1), 100.0) // Price: 10000

	as, err := NewAccountingSystem(ledger, market, "USD")
	if err != nil {
		t.Fatalf("NewAccountingSystem() error = %v", err)
	}

	// Period where ZERO has no activity and no change in price
	report, err := as.CalculateGains(date.Range{From: date.New(2025, time.March, 1), To: date.New(2025, time.March, 31)}, AverageCost)
	if err != nil {
		t.Fatalf("CalculateGains() error = %v", err)
	}

	if len(report.Securities) != 0 {
		t.Errorf("Expected 0 securities in report, got %d: %v", len(report.Securities), report.Securities)
	}
}
