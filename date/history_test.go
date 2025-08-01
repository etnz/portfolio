package date

import "testing"

func TestAppend(t *testing.T) {
	h := new(History[string])
	d1, v1 := New(2025, 07, 01), "25 Jul 1"
	d2, v2 := New(2024, 07, 01), "24 Jul 1"

	// Test is about appending two values in reverse order and checking that everything is
	// as expected at every step of the way.

	if h.Len() != 0 {
		t.Errorf("History.Len() = %v want 0", h.Len())
	}

	h.Append(d1, v1)
	if h.Len() != 1 {
		t.Errorf("Append(d1, v1).Len() = %v want 1", h.Len())
	}

	h.Append(d2, v2)
	if h.Len() != 2 {
		t.Errorf("Append(d2, v2).Len() = %v want 2", h.Len())
	}

	if h.days[1] != d1 {
		t.Errorf("history[1].day = %v want %v", h.days[0], d1)
	}
	if h.days[0] != d2 {
		t.Errorf("history[0].day = %v want %v", h.days[1], d2)
	}
	if h.values[1] != v1 {
		t.Errorf("history[1].value = %v want %v", h.values[0], v1)
	}
	if h.values[0] != v2 {
		t.Errorf("history[0].value = %v want %v", h.values[1], v2)
	}

}
