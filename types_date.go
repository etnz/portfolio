package portfolio

import (
	"encoding/json"
	"fmt"
	"regexp"
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

