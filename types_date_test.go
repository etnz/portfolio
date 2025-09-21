package portfolio

import (
	"encoding/json"
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

func TestDate_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected Date
		wantErr  bool
	}{
		{
			name:     "Zero Date from empty string",
			json:     `""`,
			expected: Date{},
			wantErr:  false,
		},
		{
			name:     "Non-Zero Date",
			json:     `"2024-05-21"`,
			expected: NewDate(2024, 5, 21),
			wantErr:  false,
		},
		{
			name:    "Invalid Date",
			json:    `"not-a-date"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var d Date
			err := json.Unmarshal([]byte(tt.json), &d)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && d != tt.expected {
				t.Errorf("json.Unmarshal() got = %v, want %v", d, tt.expected)
			}
		})
	}
}

func TestDate_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		date     Date
		expected string
		wantErr  bool
	}{
		{
			name:     "Zero Date",
			date:     Date{},
			expected: `""`,
			wantErr:  false,
		},
		{
			name:     "Non-Zero Date",
			date:     NewDate(2024, 5, 21),
			expected: `"2024-05-21"`,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if string(got) != tt.expected {
				t.Errorf("json.Marshal() = %s, want %s", got, tt.expected)
			}
		})
	}
}
