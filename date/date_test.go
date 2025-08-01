package date

import "testing"

// TestTime assert that the time() is cannonical and gives comparable times.
func TestTime(t *testing.T) {
	d1 := New(2025, 7, 31)
	d2 := New(2025, 7, 31)

	if d1.time() != d2.time() {
		// Note that usually time.Time are not comparable (there is a pointer for the timezone) this
		// tests also checks that the property remain true
		t.Errorf("invalid time() function same day gives two different time")
	}
}
