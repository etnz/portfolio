package date

import (
	"iter"
	"slices"
	"sort"
)

type History[T any] struct {
	days   []Date
	values []T
}

// Clear removes all items from the history.
func (h *History[T]) Clear() {
	h.days = h.days[:0]
	h.values = h.values[:0]
}

// Len return the number of items in the history.
func (h *History[T]) Len() int { return len(h.days) }

// chronological is a private implementation to make this history chronologically sorted.
type chronological[T any] struct{ *History[T] }

func (s chronological[T]) Less(i, j int) bool { return s.days[i].time().Before(s.days[j].time()) }

func (s chronological[T]) Swap(i, j int) {
	s.days[i], s.days[j] = s.days[j], s.days[i]
	s.values[i], s.values[j] = s.values[j], s.values[i]
}

// sort history in chronological order.
func (h *History[T]) sort() { sort.Sort(chronological[T]{h}) }

// Append a point in the history.
func (h *History[T]) Append(on Date, q T) *History[T] {
	if i := slices.Index(h.days, on); i >= 0 {
		// Found a point at that exact same instant.
		// We choose to replace, because it will give higher priority to the last data
		h.values[i] = q
		return h
	} else {
		// We need to increase the memory first.
		h.days, h.values = append(h.days, on), append(h.values, q)
	}
	h.sort()
	return h
}

// Values return an iterator over all values in the history.
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
