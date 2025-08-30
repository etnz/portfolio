package date

import (
	"encoding/json"
	"fmt"
	"iter"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const readDateFormat = "2006-1-2" // Permissive read date format (allows single-digit month/day).

// DateFormat is the format used to represent dates as strings in ISO-8601 format.
const DateFormat = "2006-01-02" // write date format

const Day = 24 * time.Hour

// Date represents a date with day-level granularity.
type Date struct {
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

// Format returns a textual representation of the date value formatted according to the layout defined by the argument.
//
//	See the documentation for the [time.Format].
func (d Date) Format(format string) string { return d.time().Format(format) }

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

var (
	relativeDateRE = regexp.MustCompile(`^([+-])(\d+)([dwmqy])$`)
	monthDayDateRE = regexp.MustCompile(`^(?:(\d+)-)?(\d+)$`)
)

// Parse parses a Date from a string. It is lenient and accepts formats like "2025-7-1".
func Parse(str string) (Date, error) {
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
			return New(today.Year(), today.Month()+time.Month(num), today.Day()), nil
		case "q":
			return New(today.Year(), today.Month()+time.Month(num*3), today.Day()), nil
		case "y":
			return New(today.Year()+num, today.Month(), today.Day()), nil
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
			return New(year, month, 0), nil
		}
		return New(year, month, day), nil
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
	return New(on.Date()), nil
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
	// Keep this parsing strict, as it's for data files.
	// But not too strict, also supports 2025-7-1
	on, err := time.Parse(readDateFormat, str)
	if err != nil {
		return fmt.Errorf("invalid date %q in data file, want format %q: %w", str, DateFormat, err)
	}
	*j = New(on.Date())
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
func Iterate[T float32 | float64](histories ...History[T]) iter.Seq[Date] {
	dates := make([][]Date, 0, len(histories))
	for _, h := range histories {
		dates = append(dates, h.days)
	}
	return iterate(dates...)
}

// Range represents a range of dates.
type Range struct{ From, To Date }
