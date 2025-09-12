package portfolio

import (
	"encoding/json"
	"fmt"
	"iter"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

const readDateFormat = "2006-1-2" // Permissive read date format (allows single-digit month/day).

// DateFormat is the format used to represent dates as strings in ISO-8601 format.
const DateFormat = "2006-01-02" // write date format
const DatetimeFormat = time.RFC3339

const Day = 24 * time.Hour

// Date represents a date with day-level granularity.
type Date struct {
	y int        // year
	m time.Month // month
	d int        // day
}

// NewDate returns a normalized Date for the given year, month, and day.
func NewDate(year int, month time.Month, day int) Date {
	d := Date{year, month, day}
	d.y, d.m, d.d = d.time().Date()
	return d
}

// Year returns current year.
func (d Date) Year() int { return d.y }

// Month returns the month of the date.
func (d Date) Month() time.Month { return d.time().Month() }

// Day returns current day of the month.
func (d Date) Day() int { return d.d }

// String format the date in date RFC3339
func (d Date) String() string { return d.time().Format(DateFormat) }

// Full format the date in date-time RFC3339
func (d Date) Full() string { return d.time().Format(DatetimeFormat) }

// IsZero returns true if the date is the zero value.
func (d Date) IsZero() bool {
	return d.y == 0 && d.m == 0 && d.d == 0
}

func (d Date) IsToday() bool {
	return d == Today()
}

// Weekday returns the day of the week for the date.
func (d Date) Weekday() time.Weekday { return d.time().Weekday() }

// ISOWeek returns the ISO 8601 year and week number in which d occurs.
func (d Date) ISOWeek() (year, week int) { return d.time().ISOWeek() }

// time returns a time.Time that is a canonical representation of that day (at midnight UTC).
func (d Date) time() time.Time { return time.Date(d.y, d.m, d.d, 0, 0, 0, 0, time.UTC) }

// Format returns a textual representation of the date value formatted according to the layout defined by the argument.
//
//	See the documentation for the [time.Format].
func (d Date) Format(format string) string { return d.time().Format(format) }

// Before reports whether the day d is before x.
func (d Date) Before(x Date) bool { return d.time().Before(x.time()) }

// After reports whether the day d is after x.
func (d Date) After(x Date) bool { return d.time().After(x.time()) }

// Today returns the current date.
func Today() Date { return NewDate(time.Now().Date()) }

// Add returns a new Date with the given number of days added.
func (d Date) Add(i int) Date { return NewDate(d.y, d.m, d.d+i) }

// AddMonth returns a new Date with the given number of days added.
func (d Date) AddMonth(i int) Date { return NewDate(d.y, d.m+time.Month(i), d.d) }

// StartOf returns the date of begining of a given period
func (d Date) StartOf(period Period) Date {
	switch period {
	case Daily:
		return d
	case Weekly:
		weekday := d.Weekday() // time.Sunday = 0, ..., time.Saturday = 6
		offset := int(weekday - time.Monday)
		for offset < 0 {
			offset += 7
		}
		return d.Add(-offset)
	case Monthly:
		return NewDate(d.Year(), d.Month(), 1)
	case Quarterly:
		quarter := (d.Month() - 1) / 3
		startMonth := time.Month(quarter*3 + 1)
		return NewDate(d.Year(), startMonth, 1)
	case Yearly:
		return NewDate(d.Year(), time.January, 1)
	default:
		panic("unknown period")
	}
}

// EndOf returns the date of end of a given period
func (d Date) EndOf(period Period) Date {
	switch period {
	case Daily:
		return d
	case Weekly:
		weekday := d.Weekday() // time.Sunday = 0, ..., time.Saturday = 6
		offset := int(7 - weekday)
		for offset >= 7 {
			offset -= 7
		}
		return d.Add(offset)
	case Monthly:
		return NewDate(d.Year(), d.Month()+1, 0)
	case Quarterly:
		quarter := (d.Month() - 1) / 3          // in [0..3]
		endMonth := time.Month(quarter*3 + 3)   // in [1..12] hence the +3
		return NewDate(d.Year(), endMonth+1, 0) // last is next month on the day 0
	case Yearly:
		return NewDate(d.Year()+1, time.January, 0)
	default:
		panic("unknown period")
	}
}

var (
	relativeDateRE = regexp.MustCompile(`^([+-])(\d+)([dwmqy])$`)
	monthDayDateRE = regexp.MustCompile(`^(?:(\d+)-)?(\d+)$`)
)

// ParseDate parses a Date from a string. It is lenient and accepts formats like "2025-7-1".
func ParseDate(str string) (Date, error) {
	str = strings.TrimSpace(str)

	// Handle "0d" as a special case for today
	if str == "0d" {
		return Today(), nil
	}

	// Relative Duration Format (e.g., -1d, +2w) - sign is mandatory for non-zero
	if match := relativeDateRE.FindStringSubmatch(str); match != nil {
		sign := match[1]
		numStr := match[2]
		unit := match[3]

		num, err := strconv.Atoi(numStr)
		if err != nil {
			// This should not happen given the regex
			return Date{}, fmt.Errorf("invalid number in relative date %q: %w", str, err)
		}

		if sign == "-" {
			num = -num
		}

		today := Today()
		switch unit {
		case "d":
			return today.Add(num), nil
		case "w":
			return today.Add(num * 7), nil
		case "m":
			return NewDate(today.Year(), today.Month()+time.Month(num), today.Day()), nil
		case "q":
			return NewDate(today.Year(), today.Month()+time.Month(num*3), today.Day()), nil
		case "y":
			return NewDate(today.Year()+num, today.Month(), today.Day()), nil
		}
	}

	// [MM-]DD Format (e.g., 27, 8-27, 0, 8-0, 0-15)
	if match := monthDayDateRE.FindStringSubmatch(str); match != nil {
		monthStr := match[1]
		dayStr := match[2]

		day, err := strconv.Atoi(dayStr)
		if err != nil {
			// This should not happen given the regex
			return Date{}, fmt.Errorf("invalid day in date %q: %w", str, err)
		}

		today := Today()
		year := today.Year()
		month := today.Month()

		if monthStr != "" {
			m, err := strconv.Atoi(monthStr)
			if err != nil {
				// This should not happen given the regex
				return Date{}, fmt.Errorf("invalid month in date %q: %w", str, err)
			}
			if m == 0 {
				year--
				month = time.December
			} else {
				month = time.Month(m)
			}
		}

		if day == 0 {
			// last day of previous month
			return NewDate(year, month, 0), nil
		}
		return NewDate(year, month, day), nil
	}

	// Standard ISO Format (Fallback)
	on, err := time.Parse(readDateFormat, str)
	// We use a slightly more permisive format for read, to support 2025-7-1 instead of 2025-07-01
	if err != nil {
		// try the long format
		on, err = time.Parse("2006-01-02T15:04:05.000-0700", str)
	}
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q want format %q: %w", str, readDateFormat, err)
	}
	return NewDate(on.Date()), nil
}

// MustParse is like Parse but panics on error.
func MustParse(str string) Date {
	d, err := ParseDate(str)
	if err != nil {
		panic(err.Error())
	}
	return d
}

// UnmarshalJSON implements the json specific way to unmarshall a date from a json string.
func (j *Date) UnmarshalJSON(bytes []byte) error {
	var str string
	if err := json.Unmarshal(bytes, &str); err != nil {
		return err
	}
	// Keep this parsing strict, as it's for data files.
	// But not too strict, also supports 2025-7-1
	on, err := time.Parse(readDateFormat, str)
	if err != nil {
		return fmt.Errorf("invalid date %q in data file, want format %q: %w", str, DateFormat, err)
	}
	*j = NewDate(on.Date())
	return nil
}
func (j Date) MarshalJSON() ([]byte, error) {
	str := j.String()
	return json.Marshal(&str)
}

// check that a Date pointer is a valid json marshall/unmarshaller type.
var _ json.Marshaler = (*Date)(nil)
var _ json.Unmarshaler = (*Date)(nil)

// iterate returns an iterator over all unique, sorted dates from multiple series of dates.
func iterate(series ...[]Date) iter.Seq[Date] {
	return func(yield func(Date) bool) {
		indexes := make([]int, len(series))
		// find the reached mins
		times := make([]Date, 0, len(series))
		for {
			times = times[:0] //empty the slice again
			for i, index := range indexes {
				if index < len(series[i]) {
					on := series[i][index]
					times = append(times, on)
				}
			}
			if len(times) == 0 {
				// All timeseries have been consumed, exit.
				return
			}
			// there are some remaining values:
			var m Date
			if len(times) > 0 {
				m = times[0]
				for _, t := range times {
					if t.Before(m) {
						m = t
					}
				}
			}
			// now extract the ones that are equals to the min
			for i, index := range indexes {
				if index >= len(series[i]) {
					continue
				}
				if on := series[i][index]; on == m {
					// Updates and consume this value
					indexes[i]++
				}
			}
			if !yield(m) {
				return
			}
		}
	}
}

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

// Iterate returns an iterator over all unique, sorted dates from multiple History objects.
func Iterate[T float32 | float64](histories ...History[T]) iter.Seq[Date] {
	dates := make([][]Date, 0, len(histories))
	for _, h := range histories {
		dates = append(dates, h.days)
	}
	return iterate(dates...)
}

// History stores a chronological series of values, each associated with a specific date.
// It ensures that dates are unique and the series is always sorted.
type History[T float32 | float64 | string] struct {
	days   []Date
	values []T
}

// Latest returns the latest date and value in the history.
// If the history is empty, it returns zero value.
func (h *History[T]) Latest() (day Date, value T) {
	last := len(h.days) - 1
	if last < 0 {
		return Date{}, *new(T) // return zero value of T
	}
	return h.days[last], h.values[last]
}

// Clear removes all items from the history.
func (h *History[T]) Clear() {
	h.days = h.days[:0]
	h.values = h.values[:0]
}

// Len returns the number of items in the history.
func (h *History[T]) Len() int { return len(h.days) }

// chronological is a private implementation to make this history chronologically sorted.
type chronological[T float32 | float64 | string] struct{ *History[T] }

func (s chronological[T]) Less(i, j int) bool { return s.days[i].time().Before(s.days[j].time()) }

func (s chronological[T]) Swap(i, j int) {
	s.days[i], s.days[j] = s.days[j], s.days[i]
	s.values[i], s.values[j] = s.values[j], s.values[i]
}

// sort sorts the history in chronological order.
func (h *History[T]) sort() { sort.Sort(chronological[T]{h}) }

// Append adds a point to the history.
//
// Existing value at that date are overwritten.
func (h *History[T]) Append(on Date, q T) *History[T] {
	if i := slices.Index(h.days, on); i >= 0 {
		// Found a point at that exact same instant.
		// We choose to replace, because it will give higher priority to the last data
		h.values[i] = q
		return h
	}
	// We need to increase the memory first.
	h.days, h.values = append(h.days, on), append(h.values, q)
	h.sort()
	return h
}

// AppendAdd adds a point to the history.
//
// Existing value is added.
func (h *History[T]) AppendAdd(on Date, q T) *History[T] {
	if i := slices.Index(h.days, on); i >= 0 {
		// Found a point at that exact same instant.
		// We choose to replace, because it will give higher priority to the last data
		h.values[i] += q
		return h
	}
	// We need to increase the memory first.
	h.days, h.values = append(h.days, on), append(h.values, q)
	h.sort()
	return h
}

// Values returns an iterator over all date/value pairs in the history, in chronological order.
func (h *History[T]) Values() iter.Seq2[Date, T] {
	return func(yield func(Date, T) bool) {
		for i, on := range h.days {
			if !yield(on, h.values[i]) {
				return
			}
		}
	}
}

// Get returns the value at 'day' and true or zero value and false.
func (f *History[T]) Get(day Date) (T, bool) {
	var value T
	i := slices.Index(f.days, day)
	if i >= 0 {
		return f.values[i], true
	}
	return value, false
}

// ValueAsOf returns the value on a given day, or the most recent value before it.
// It returns the value and true if found, otherwise it returns the zero value and false.
func (h *History[T]) ValueAsOf(day Date) (T, bool) {
	// The days slice is sorted, so we can use binary search.
	i, found := slices.BinarySearchFunc(h.days, day, func(d, t Date) int {
		if d.After(t) {
			return 1
		}
		if d.Before(t) {
			return -1
		}
		return 0
	})

	if found {
		return h.values[i], true
	}

	// Not found. `i` is the index where `day` would be inserted.
	// The value we want is at `i-1`, which is the last entry before the target date.
	if i == 0 {
		var zero T
		return zero, false // No date on or before the given day.
	}
	return h.values[i-1], true
}
