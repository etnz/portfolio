package date

import "time"

// StartOfWeek returns the date of the Monday of the week containing 'd'.
func StartOfWeek(d Date) Date {
	weekday := d.Weekday() // time.Sunday = 0, ..., time.Saturday = 6
	offset := int(weekday - time.Monday)
	if offset < 0 {
		offset += 7
	}
	return d.Add(-offset)
}

// StartOfMonth returns the first day of the month containing 'd'.
func StartOfMonth(d Date) Date {
	return New(d.Year(), d.Month(), 1)
}

// StartOfQuarter returns the first day of the quarter containing 'd'.
func StartOfQuarter(d Date) Date {
	quarter := (d.Month() - 1) / 3
	startMonth := time.Month(quarter*3 + 1)
	return New(d.Year(), startMonth, 1)
}

// StartOfYear returns the first day of the year containing 'd'.
func StartOfYear(d Date) Date {
	return New(d.Year(), time.January, 1)
}

// NewRangeFrom returns a new date range from an end date and a period string.
func NewRangeFrom(end Date, period string) Range {
	var start Date
	switch period {
	case "day":
		start = end
	case "week":
		start = StartOfWeek(end)
	case "month":
		start = StartOfMonth(end)
	case "quarter":
		start = StartOfQuarter(end)
	case "year", "": // default to year
		start = StartOfYear(end)
	default:
		// maybe we should return an error here
		start = StartOfYear(end)
	}
	return Range{From: start, To: end}
}
