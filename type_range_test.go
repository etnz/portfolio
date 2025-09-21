package portfolio

import (
	"reflect"
	"slices"
	"testing"
)

func TestRange_Periods(t *testing.T) {
	tests := []struct {
		name     string
		r        Range
		p        Period
		expected []Range
	}{
		{
			name: "Weekly periods over two weeks",
			r:    NewRange(NewDate(2024, 1, 10), NewDate(2024, 1, 17)), // Wednesday to Wednesday
			p:    Weekly,
			expected: []Range{
				NewRange(NewDate(2024, 1, 8), NewDate(2024, 1, 14)),
				NewRange(NewDate(2024, 1, 15), NewDate(2024, 1, 21)),
			},
		},
		{
			name: "Monthly periods over parts of three months",
			r:    NewRange(NewDate(2024, 2, 15), NewDate(2024, 4, 10)),
			p:    Monthly,
			expected: []Range{
				NewRange(NewDate(2024, 2, 1), NewDate(2024, 2, 29)),
				NewRange(NewDate(2024, 3, 1), NewDate(2024, 3, 31)),
				NewRange(NewDate(2024, 4, 1), NewDate(2024, 4, 30)),
			},
		},
		{
			name: "Daily periods",
			r:    NewRange(NewDate(2024, 1, 1), NewDate(2024, 1, 3)),
			p:    Daily,
			expected: []Range{
				NewRange(NewDate(2024, 1, 1), NewDate(2024, 1, 1)),
				NewRange(NewDate(2024, 1, 2), NewDate(2024, 1, 2)),
				NewRange(NewDate(2024, 1, 3), NewDate(2024, 1, 3)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := slices.Collect(tt.r.Periods(tt.p))
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("Range.Periods() = %v, want %v", got, tt.expected)
			}
		})
	}
}
