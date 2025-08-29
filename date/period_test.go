package date

import (
	"testing"
	"time"
)

func TestStartOfWeek(t *testing.T) {
	testCases := []struct {
		name string
		in   Date
		want Date
	}{
		{
			name: "A Monday",
			in:   New(2024, time.July, 1), // Monday
			want: New(2024, time.July, 1),
		},
		{
			name: "A Wednesday",
			in:   New(2024, time.July, 3), // Wednesday
			want: New(2024, time.July, 1),
		},
		{
			name: "A Sunday",
			in:   New(2024, time.July, 7), // Sunday
			want: New(2024, time.July, 1),
		},
		{
			name: "A Sunday at the beginning of a month",
			in:   New(2024, time.September, 1), // Sunday
			want: New(2024, time.August, 26),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := StartOfWeek(tc.in)
			if got != tc.want {
				t.Errorf("StartOfWeek(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestStartOfMonth(t *testing.T) {
	testCases := []struct {
		name string
		in   Date
		want Date
	}{
		{
			name: "First day of month",
			in:   New(2024, time.July, 1),
			want: New(2024, time.July, 1),
		},
		{
			name: "Middle of month",
			in:   New(2024, time.July, 15),
			want: New(2024, time.July, 1),
		},
		{
			name: "Last day of month",
			in:   New(2024, time.July, 31),
			want: New(2024, time.July, 1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := StartOfMonth(tc.in)
			if got != tc.want {
				t.Errorf("StartOfMonth(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestStartOfQuarter(t *testing.T) {
	testCases := []struct {
		name string
		in   Date
		want Date
	}{
		{name: "Q1 (Jan)", in: New(2024, time.January, 15), want: New(2024, time.January, 1)},
		{name: "Q1 (Mar)", in: New(2024, time.March, 31), want: New(2024, time.January, 1)},
		{name: "Q2 (Apr)", in: New(2024, time.April, 1), want: New(2024, time.April, 1)},
		{name: "Q2 (Jun)", in: New(2024, time.June, 20), want: New(2024, time.April, 1)},
		{name: "Q3 (Jul)", in: New(2024, time.July, 10), want: New(2024, time.July, 1)},
		{name: "Q3 (Sep)", in: New(2024, time.September, 30), want: New(2024, time.July, 1)},
		{name: "Q4 (Oct)", in: New(2024, time.October, 5), want: New(2024, time.October, 1)},
		{name: "Q4 (Dec)", in: New(2024, time.December, 31), want: New(2024, time.October, 1)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := StartOfQuarter(tc.in)
			if got != tc.want {
				t.Errorf("StartOfQuarter(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestStartOfYear(t *testing.T) {
	testCases := []struct {
		name string
		in   Date
		want Date
	}{
		{
			name: "First day of year",
			in:   New(2024, time.January, 1),
			want: New(2024, time.January, 1),
		},
		{
			name: "Middle of year",
			in:   New(2024, time.July, 15),
			want: New(2024, time.January, 1),
		},
		{
			name: "Last day of year",
			in:   New(2024, time.December, 31),
			want: New(2024, time.January, 1),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got := StartOfYear(tc.in)
			if got != tc.want {
				t.Errorf("StartOfYear(%s) = %s, want %s", tc.in, got, tc.want)
			}
		})
	}
}
