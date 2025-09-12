package cmd

import (
	"testing"
	"text/template"
	"time"

	"github.com/etnz/portfolio"
)

// EUR is a helper for test to create euro money from const
func EUR(v float64) portfolio.Money { return portfolio.M(v, "EUR") }

// USD is a helper for test to create usd money from const
func USD(v float64) portfolio.Money { return portfolio.M(v, "USD") }

// NO is a helper for test to create money from const wit no currency set
func NO(v float64) portfolio.Money { return portfolio.M(v, "") }

// Q is a helper for test to create Quantity from const
func Q(v float64) portfolio.Quantity { return portfolio.Q(v) }

func newDate(t *testing.T, s string) portfolio.Date {
	timeVal, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatal(err)
	}
	return portfolio.NewDate(timeVal.Date())
}

func newTx(on portfolio.Date) portfolio.Transaction {
	return portfolio.NewBuy(on, "", "AAPL", Q(0), USD(0))
}

func TestGeneratePeriods(t *testing.T) {
	tests := []struct {
		name          string
		transactions  []portfolio.Transaction
		wantRanges    int
		wantDaily     int
		wantWeekly    int
		wantMonthly   int
		wantQuarterly int
		wantYearly    int
	}{
		{
			name:          "empty ledger",
			transactions:  []portfolio.Transaction{},
			wantRanges:    0,
			wantDaily:     0,
			wantWeekly:    0,
			wantMonthly:   0,
			wantQuarterly: 0,
			wantYearly:    0,
		},
		{
			name: "single day",
			transactions: []portfolio.Transaction{
				newTx(newDate(t, "2025-08-15")),
			},
			wantDaily:     1,
			wantWeekly:    1,
			wantMonthly:   1,
			wantQuarterly: 1,
			wantYearly:    1,
		},
		{
			name: "multi-week, single-month",
			transactions: []portfolio.Transaction{
				newTx(newDate(t, "2025-08-10")),
				newTx(newDate(t, "2025-08-25")),
			},
			wantDaily:     16,
			wantWeekly:    4, // W32, W33, W34, W35
			wantMonthly:   1,
			wantQuarterly: 1,
			wantYearly:    1,
		},
		{
			name: "cross-year boundary",
			transactions: []portfolio.Transaction{
				newTx(newDate(t, "2024-12-15")),
				newTx(newDate(t, "2025-01-15")),
			},
			wantDaily:     32,
			wantWeekly:    6, // W51, W52 (2024), W1, W2, W3 (2025)
			wantMonthly:   2,
			wantQuarterly: 2,
			wantYearly:    2,
		},
		{
			name: "full year",
			transactions: []portfolio.Transaction{
				newTx(newDate(t, "2023-01-01")),
				newTx(newDate(t, "2023-12-31")),
			},
			wantDaily:     365,
			wantWeekly:    53,
			wantMonthly:   12,
			wantQuarterly: 4,
			wantYearly:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ledger := portfolio.NewLedger()
			for _, tx := range tt.transactions {
				ledger.Append(tx)
			}

			ranges, err := generatePeriods(ledger.OldestTransactionDate(), ledger.NewestTransactionDate())
			if err != nil {
				t.Fatalf("generatePeriods() error = %v", err)
			}

			// Filter out ranges that are not standard periods (e.g., the ones created by NewRange(startDate, periodType))
			// and count them by type.
			var daily, weekly, monthly, quarterly, yearly int
			for _, r := range ranges {
				p, ok := r.Period()
				if !ok {
					continue
				}
				switch p {
				case portfolio.Daily:
					daily++
				case portfolio.Weekly:
					weekly++
				case portfolio.Monthly:
					monthly++
				case portfolio.Quarterly:
					quarterly++
				case portfolio.Yearly:
					yearly++
				}
			}

			if tt.wantDaily != daily {
				t.Errorf("generatePeriods() got %d daily ranges, want %d", daily, tt.wantDaily)
			}
			if tt.wantWeekly != weekly {
				t.Errorf("generatePeriods() got %d weekly ranges, want %d", weekly, tt.wantWeekly)
			}
			if tt.wantMonthly != monthly {
				t.Errorf("generatePeriods() got %d monthly ranges, want %d", monthly, tt.wantMonthly)
			}
			if tt.wantQuarterly != quarterly {
				t.Errorf("generatePeriods() got %d quarterly ranges, want %d", quarterly, tt.wantQuarterly)
			}
			if tt.wantYearly != yearly {
				t.Errorf("generatePeriods() got %d yearly ranges, want %d", yearly, tt.wantYearly)
			}
		})
	}
}

func TestRenderFrontMatter(t *testing.T) {
	tests := []struct {
		name     string
		template string
		task     reportTask
		want     string
		wantErr  bool
	}{
		{
			name:     "basic template",
			template: "---\ntitle: {{.Report}} Report for {{.Period.Identifier}}\n---",
			task:     reportTask{Report: "review", Period: portfolio.Daily.Range(newDate(t, "2025-01-01"))},
			want:     "---\ntitle: review Report for 2025-01-01\n---",
			wantErr:  false,
		},
		{
			name: "api",
			template: `
{{.Report}}: The type of report (e.g., "review", "holding").
{{.Period.From}}: The start date of the report.
{{.Period.To}}: The end date of the report.
{{.Period.To.Full}}: The end date in RFC3339 format.
{{.Period.Name}}: A human-readable name for the period (e.g., "Daily", "Weekly", "Monthly").
{{.Period.To.Format "January 06"}}: A formatted string of the end date.`,
			task: reportTask{Report: "review", Period: portfolio.Weekly.Range(newDate(t, "2025-01-01"))},
			want: `
review: The type of report (e.g., "review", "holding").
2024-12-30: The start date of the report.
2025-01-05: The end date of the report.
2025-01-05T00:00:00Z: The end date in RFC3339 format.
weekly: A human-readable name for the period (e.g., "Daily", "Weekly", "Monthly").
January 25: A formatted string of the end date.`,
			wantErr: false,
		},
		{
			name:     "empty template",
			template: "",
			task:     reportTask{Report: "review", Period: portfolio.Daily.Range(newDate(t, "2025-01-01"))},
			want:     "",
			wantErr:  false,
		},
		{
			name:     "template with error",
			template: "{{.NonExistentField}}",
			task:     reportTask{Report: "review", Period: portfolio.Daily.Range(newDate(t, "2025-01-01"))},
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tpl, err := template.New("test").Parse(tt.template)
			if err != nil && !tt.wantErr {
				t.Fatalf("failed to parse template: %v", err)
			}

			got, err := renderFrontMatter(tpl, tt.task)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderFrontMatter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("renderFrontMatter() got = %v, want %v", got, tt.want)
			}
		})
	}
}
