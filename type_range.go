package portfolio

import (
	"fmt"
	"iter"
	"time"
)

// Range represents a range of dates.
type Range struct{ From, To Date }

// NewRange creates a new date range. If 'from' is after 'to', they are swapped.
func NewRange(from, to Date) Range {
	if from.After(to) {
		from, to = to, from
	}
	return Range{From: from, To: to}
}

// Contains return true date is included in the range (boundaries included)
func (r Range) Contains(date Date) bool { return (!date.Before(r.From) && !date.After(r.To)) }

// Days returns an iterator that yields each date within the range, inclusive.
func (r Range) Days() iter.Seq[Date] {
	return func(yield func(Date) bool) {
		for d := r.From; !d.After(r.To); d = d.Add(1) {
			if !yield(d) {
				return
			}
		}
	}
}

// Periods returns an iterator that yields each sequential range of a given
// period 'p' that contains at least one day within the original range 'r'.
func (r Range) Periods(p Period) iter.Seq[Range] {
	return func(yield func(Range) bool) {
		// Start from the beginning of the original range.
		for current := r.From; !current.After(r.To); {
			// Get the full period range containing the current date.
			periodRange := p.Range(current)
			if !yield(periodRange) {
				return
			}
			// Move to the day after the end of the yielded period to start the next iteration.
			current = periodRange.To.Add(1)
		}
	}
}

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
	p, _ := r.Period()
	return p.Name()
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
		return r.From.Format("2006-January")
	case Quarterly:
		return fmt.Sprintf("%d-Q%d", r.From.Year(), (r.From.Month()-1)/3+1)
	case Yearly:
		return r.From.Format("2006")
	default:
		panic("unknown period")
	}
}
