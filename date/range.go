package date

import (
	"fmt"
	"time"
)

// Range represents a range of dates.
type Range struct{ From, To Date }

// NewRange return a well known period
func NewRange(d Date, period Period) Range {
	return Range{From: d.StartOf(period), To: d.EndOf(period)}
}

// Contains return true date is included in the range (boundaries included)
func (r Range) Contains(date Date) bool { return (!date.Before(r.From) && !date.After(r.To)) }

// return the period of this range if it's a standard one.
func (r Range) Period() (p Period, ok bool) {
	switch {
	case r.From == r.To:
		return Daily, true
	case r.From.Weekday() == time.Monday && r.From.EndOf(Weekly) == r.To:
		return Weekly, true
	case r.From.Day() == 1 && r.From.EndOf(Monthly) == r.To:
		return Monthly, true
	case r.From.StartOf(Quarterly) == r.From && r.From.EndOf(Quarterly) == r.To:
		return Quarterly, true
	case r.From.StartOf(Yearly) == r.From && r.From.EndOf(Yearly) == r.To:
		return Yearly, true
	default:
		return Daily, false
	}
}

// Name the period range
func (r Range) Name() string {
	p, ok := r.Period()
	if ok {
		return p.String()
	}
	return "special"
}

// Identifier compute a unique identifier for the Range.
// If the period is defined, use a short insighful name
func (r Range) Identifier() string {

	p, ok := r.Period()
	if !ok {
		return fmt.Sprintf("%s_%s", r.From, r.To)
	}

	switch p {
	case Daily:
		return r.From.String()
	case Weekly:
		_, week := r.From.ISOWeek()
		return fmt.Sprintf("%d-W%02d", r.From.Year(), week)
	case Monthly:
		return r.From.Format("2006-01")
	case Quarterly:
		return fmt.Sprintf("%d-Q%d", r.From.Year(), (r.From.Month()-1)/3+1)
	case Yearly:
		return r.From.Format("2006")
	default:
		panic("unknown period")
	}

}
