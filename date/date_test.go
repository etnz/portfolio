package date

import (
	"fmt"
	"testing"
	"time"
)

// TestTime assert that the time() is cannonical and gives comparable times.
func TestTime(t *testing.T) {
	d1 := New(2025, 7, 31)
	d2 := New(2025, 7, 31)

	if d1.time() != d2.time() {
		// Note that usually time.Time are not comparable (there is a pointer for the timezone) this
		// tests also checks that the property remain true
		t.Errorf("invalid time() function same day gives two different time")
	}
}

func TestParse(t *testing.T) {
	today := Today()
	currentYear := today.Year()
	currentMonth := today.Month()

	tests := []struct {
		input    string
		expected Date
		err      bool
	}{
		// Standard ISO Format (Fallback)
		{"2025-01-15", New(2025, time.January, 15), false},
		{"2025-7-1", New(2025, time.July, 1), false},
		{"invalid-date", Date{}, true},

		// Relative Duration Format
		{"-1d", today.Add(-1), false},
		{"+1d", today.Add(1), false},
		{"1d", Date{}, true},
		{"-0d", today, false},
		{"+0d", today, false},
		{"-2w", today.Add(-14), false},
		{"+1m", New(currentYear, currentMonth+1, today.Day()), false},
		{"-3q", New(currentYear, currentMonth-9, today.Day()), false},
		{"+1y", New(currentYear+1, currentMonth, today.Day()), false},
		{"-1y", New(currentYear-1, currentMonth, today.Day()), false},

		// [MM-]DD Format
		{"27", New(currentYear, currentMonth, 27), false},
		{fmt.Sprintf("%d-27", currentMonth), New(currentYear, currentMonth, 27), false},
		{"0", New(currentYear, currentMonth, 0), false},                               // Last day of previous month
		{fmt.Sprintf("%d-0", currentMonth), New(currentYear, currentMonth, 0), false}, // Last day of previous month
		{"1-15", New(currentYear, time.January, 15), false},
		{"0-15", New(currentYear-1, time.December, 15), false},
		{"1-0", New(currentYear-1, time.December, 31), false}, // Last day of previous year
		{"0-0", New(currentYear-1, time.November, 30), false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("Parse(%q) error = %v, wantErr %v", tt.input, err, tt.err)
				return
			}
			if !tt.err && got != tt.expected {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
