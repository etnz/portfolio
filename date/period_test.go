package date

import (
	"testing"
	"time"
)

func NewDailyRange(d Date) Range {
	return NewRange(d, Daily)
}
func NewWeeklyRange(d Date) Range {
	return NewRange(d, Weekly)
}
func NewMonthlyRange(d Date) Range {
	return NewRange(d, Monthly)
}
func NewQuarterlyRange(d Date) Range {
	return NewRange(d, Quarterly)
}
func NewYearlyRange(d Date) Range {
	return NewRange(d, Yearly)
}

func TestNewDailyRange(t *testing.T) {
	d := New(2025, time.September, 8)
	want := Range{From: d, To: d}
	got := NewDailyRange(d)
	if got != want {
		t.Errorf("NewDailyRange() = %v, want %v", got, want)
	}
}

func TestNewWeeklyRange(t *testing.T) {
	testCases := []struct {
		name string
		in   Date
		want Range
	}{
		{
			name: "A Wednesday",
			in:   New(2025, time.September, 10),
			want: Range{From: New(2025, time.September, 8), To: New(2025, time.September, 14)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewWeeklyRange(tc.in); got != tc.want {
				t.Errorf("NewWeeklyRange() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewMonthlyRange(t *testing.T) {
	testCases := []struct {
		name string
		in   Date
		want Range
	}{
		{
			name: "A leap year",
			in:   New(2024, time.February, 15),
			want: Range{From: New(2024, time.February, 1), To: New(2024, time.February, 29)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewMonthlyRange(tc.in); got != tc.want {
				t.Errorf("NewMonthlyRange() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewQuarterlyRange(t *testing.T) {
	testCases := []struct {
		name string
		in   Date
		want Range
	}{
		{
			name: "Q2",
			in:   New(2025, time.May, 20),
			want: Range{From: New(2025, time.April, 1), To: New(2025, time.June, 30)},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NewQuarterlyRange(tc.in); got != tc.want {
				t.Errorf("NewQuarterlyRange() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewYearlyRange(t *testing.T) {
	d := New(2025, time.September, 8)
	want := Range{From: New(2025, time.January, 1), To: New(2025, time.December, 31)}
	got := NewYearlyRange(d)
	if got != want {
		t.Errorf("NewYearlyRange() = %v, want %v", got, want)
	}
}

func TestRange_Name(t *testing.T) {
	testCases := []struct {
		name string
		in   Range
		want string
	}{
		{"Single Day", NewDailyRange(New(2025, time.September, 8)), "daily"},
		{"Standard Week", NewWeeklyRange(New(2025, time.September, 8)), "weekly"},
		{"Standard Month", NewMonthlyRange(New(2025, time.September, 1)), "monthly"},
		{"Standard Quarter", NewQuarterlyRange(New(2025, time.July, 1)), "quarterly"},
		{"Standard Year", NewYearlyRange(New(2025, time.January, 1)), "yearly"},
		{"Non-Standard Range", Range{From: New(2025, time.September, 2), To: New(2025, time.September, 10)}, "special"},
		{"Leap Year Month", NewMonthlyRange(New(2024, time.February, 1)), "monthly"},
		{"Multi Year", Range{From: New(2025, time.January, 1), To: New(2026, time.December, 31)}, "special"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Name(); got != tc.want {
				t.Errorf("Name() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRange_Identifier(t *testing.T) {
	testCases := []struct {
		name string
		in   Range
		want string
	}{
		{"Daily Identifier", NewDailyRange(New(2025, time.September, 8)), "2025-09-08"},
		{"Weekly Identifier", NewWeeklyRange(New(2025, time.September, 8)), "2025-W37"},
		{"Early Week Identifier", NewWeeklyRange(New(2025, time.January, 6)), "2025-W02"},
		{"Monthly Identifier", NewMonthlyRange(New(2025, time.September, 1)), "2025-09"},
		{"Quarterly Identifier", NewQuarterlyRange(New(2025, time.July, 1)), "2025-Q3"},
		{"Yearly Identifier", NewYearlyRange(New(2025, time.January, 1)), "2025"},
		{"Custom Range Identifier", Range{From: New(2025, time.September, 2), To: New(2025, time.September, 10)}, "2025-09-02_2025-09-10"},
		{"Eror Prone Identifier", Range{From: New(2025, time.January, 1), To: New(2026, time.December, 31)}, "2025-01-01_2026-12-31"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.in.Identifier(); got != tc.want {
				t.Errorf("Identifier() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestParsePeriod(t *testing.T) {
	testCases := []struct {
		name    string
		in      string
		want    Period
		wantErr bool
	}{
		{"Daily", "daily", Daily, false},
		{"Weekly", "weekly", Weekly, false},
		{"Monthly", "monthly", Monthly, false},
		{"Quarterly", "quarterly", Quarterly, false},
		{"Yearly", "yearly", Yearly, false},
		{"Unknown", "unknown", Daily, true},
		{"Daily", "day", Daily, false},
		{"Weekly", "week", Weekly, false},
		{"Monthly", "month", Monthly, false},
		{"Quarterly", "quarter", Quarterly, false},
		{"Yearly", "year", Yearly, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParsePeriod(tc.in)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParsePeriod() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if got != tc.want {
				t.Errorf("ParsePeriod() = %v, want %v", got, tc.want)
			}
		})
	}
}
