package portfolio

import (
	"fmt"
	"strings"
)

type Period int

func (p Period) String() string {
	switch p {
	case Daily:
		return "daily"
	case Weekly:
		return "weekly"
	case Monthly:
		return "monthly"
	case Quarterly:
		return "quarterly"
	case Yearly:
		return "yearly"
	default:
		return "periodic"
	}
}

// ToDateName returns the "-to-Date" name for the period (e.g., "Month-to-Date").
func (p Period) ToDateName() string {
	switch p {
	case Daily:
		return "Today's" // A "Day-to-Date" doesn't make much sense.
	case Weekly:
		return "Week-to-Date"
	case Monthly:
		return "Month-to-Date"
	case Quarterly:
		return "Quarter-to-Date"
	case Yearly:
		return "Year-to-Date"
	default:
		// This should be unreachable
		return p.Name() + "-to-Date"
	}
}

// Name returns the singular noun for the period (e.g., "day", "week", "month").
func (p Period) Name() string {
	switch p {
	case Daily:
		return "day"
	case Weekly:
		return "week"
	case Monthly:
		return "month"
	case Quarterly:
		return "quarter"
	case Yearly:
		return "year"
	default:
		return "Period"
	}
}

// Range returns a Range for the given period containing the date d.
func (p Period) Range(d Date) Range {
	return Range{From: d.StartOf(p), To: d.EndOf(p)}
}

const (
	Daily Period = iota
	Weekly
	Monthly
	Quarterly
	Yearly
)

func ParsePeriod(p string) (Period, error) {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "daily", "day":
		return Daily, nil
	case "weekly", "week":
		return Weekly, nil
	case "monthly", "month":
		return Monthly, nil
	case "quarterly", "quarter":
		return Quarterly, nil
	case "yearly", "year":
		return Yearly, nil
	default:
		return Daily, fmt.Errorf("unknown period %s", p)
	}
}
