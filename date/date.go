package date

import (
	"encoding/json"
	"fmt"
	"iter"
	"time"
)

const readDateFormat = "2006-1-2" // Permissive read date format (allows single-digit month/day).

// DateFormat is the format used to represent dates as strings in ISO-8601 format.
const DateFormat = "2006-01-02" // write date format

const Day = 24 * time.Hour

// Date represent a date with no lower than day granularity.
type Date struct { // Date represents a date with day-level granularity.
	y int
	m time.Month
	d int
}

// Month returns the month of the date.
func (d Date) Month() time.Month { return d.time().Month() }

// Weekday returns the day of the week for the date.
func (d Date) Weekday() time.Weekday { return d.time().Weekday() }

// ISOWeek returns the ISO 8601 year and week number in which d occurs.
func (d Date) ISOWeek() (year, week int) { return d.time().ISOWeek() }

// time returns a time.Time that is a canonical representation of that day (at midnight UTC).
func (d Date) time() time.Time { return time.Date(d.y, d.m, d.d, 0, 0, 0, 0, time.UTC) }

// New returns a normalized Date for the given year, month, and day.
func New(year int, month time.Month, day int) Date {
	d := Date{year, month, day}
	d.y, d.m, d.d = d.time().Date()
	return d
}

// Before reports whether the day d is before x.
func (d Date) Before(x Date) bool { return d.time().Before(x.time()) }

// After reports whether the day d is after x.
func (d Date) After(x Date) bool { return d.time().After(x.time()) }

// Today returns the current date.
func Today() Date { return New(time.Now().Date()) }

// Add returns a new Date with the given number of days added.
func (d Date) Add(i int) Date { return New(d.y, d.m, d.d+i) }

// Year returns current year.
func (d Date) Year() int { return d.y }

// Day returns current day of the month.
func (d Date) Day() int { return d.d }

// String format the date in its standard format.
func (d Date) String() string { return d.time().Format(DateFormat) }

// Parse parses a Date from a string. It is lenient and accepts formats like "2025-7-1".
func Parse(str string) (Date, error) {
	on, err := time.Parse(readDateFormat, str)
	// We use a slightly more permisive format for read, to support 2025-7-1 instead of 2025-07-01
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q want format %q: %w", str, readDateFormat, err)
	}
	return Date(New(on.Date())), nil
}

// MustParse is like Parse but panics on error.
func MustParse(str string) Date {
	d, err := Parse(str)
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
	d, err := Parse(str)
	if err != nil {
		return err
	}
	*j = d
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

// Iterate returns an iterator over all unique, sorted dates from multiple History objects.
func Iterate[T any](histories ...History[T]) iter.Seq[Date] {
	dates := make([][]Date, 0, len(histories))
	for _, h := range histories {
		dates = append(dates, h.days)
	}
	return iterate(dates...)
}

// TODO #2: fill and test this package see issue

// Range represents a range of dates.
type Range struct{ From, To Date }
