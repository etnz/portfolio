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
