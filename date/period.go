package date

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
		panic(fmt.Sprintf("unknown period %d", p))
	}
}

const (
	Daily Period = iota
	Weekly
	Monthly
	Quarterly
	Yearly
)

func ParsePeriod(p string) (Period, error) {
	p = strings.ToLower(p)
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
