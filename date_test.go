package portfolio

import (
	"fmt"
	"testing"
	"time"
)

// TestTime assert that the time() is cannonical and gives comparable times.
func TestTime(t *testing.T) {
	d1 := NewDate(2025, 7, 31)
	d2 := NewDate(2025, 7, 31)

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
		{"2025-01-15", NewDate(2025, time.January, 15), false},
		{"2025-7-1", NewDate(2025, time.July, 1), false},
		{"invalid-date", Date{}, true},

		// Relative Duration Format
		{"-1d", today.Add(-1), false},
		{"+1d", today.Add(1), false},
		{"1d", Date{}, true},
		{"-0d", today, false},
		{"+0d", today, false},
		{"-2w", today.Add(-14), false},
		{"+1m", NewDate(currentYear, currentMonth+1, today.Day()), false},
		{"-3q", NewDate(currentYear, currentMonth-9, today.Day()), false},
		{"+1y", NewDate(currentYear+1, currentMonth, today.Day()), false},
		{"-1y", NewDate(currentYear-1, currentMonth, today.Day()), false},

		// [MM-]DD Format
		{"27", NewDate(currentYear, currentMonth, 27), false},
		{fmt.Sprintf("%d-27", currentMonth), NewDate(currentYear, currentMonth, 27), false},
		{"0", NewDate(currentYear, currentMonth, 0), false},                               // Last day of previous month
		{fmt.Sprintf("%d-0", currentMonth), NewDate(currentYear, currentMonth, 0), false}, // Last day of previous month
		{"1-15", NewDate(currentYear, time.January, 15), false},
		{"0-15", NewDate(currentYear-1, time.December, 15), false},
		{"1-0", NewDate(currentYear-1, time.December, 31), false}, // Last day of previous year
		{"0-0", NewDate(currentYear-1, time.November, 30), false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDate(tt.input)
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

func TestAppend(t *testing.T) {
	h := new(History[string])
	d1, v1 := NewDate(2025, 07, 01), "25 Jul 1"
	d2, v2 := NewDate(2024, 07, 01), "24 Jul 1"

	// Test is about appending two values in reverse order and checking that everything is
	// as expected at every step of the way.

	if h.Len() != 0 {
		t.Errorf("History.Len() = %v want 0", h.Len())
	}

	h.Append(d1, v1)
	if h.Len() != 1 {
		t.Errorf("Append(d1, v1).Len() = %v want 1", h.Len())
	}

	h.Append(d2, v2)
	if h.Len() != 2 {
		t.Errorf("Append(d2, v2).Len() = %v want 2", h.Len())
	}

	if h.days[1] != d1 {
		t.Errorf("history[1].day = %v want %v", h.days[0], d1)
	}
	if h.days[0] != d2 {
		t.Errorf("history[0].day = %v want %v", h.days[1], d2)
	}
	if h.values[1] != v1 {
		t.Errorf("history[1].value = %v want %v", h.values[0], v1)
	}
	if h.values[0] != v2 {
		t.Errorf("history[0].value = %v want %v", h.values[1], v2)
	}

}
