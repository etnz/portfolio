package date

import (
	"iter"
	"slices"
	"sort"
)

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
