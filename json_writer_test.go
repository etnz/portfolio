package portfolio

import (
	"encoding/json"
	"testing"
)

func TestJsonObjectWriter(t *testing.T) {
	t.Run("empty object", func(t *testing.T) {
		var w jsonObjectWriter
		got, err := w.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if want := "{}"; string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("simple object", func(t *testing.T) {
		var w jsonObjectWriter
		w.Append("a", 1)
		w.Append("b", "hello")
		got, err := w.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"a":1,"b":"hello"}`
		if string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("embed object", func(t *testing.T) {
		var w jsonObjectWriter
		embedded := json.RawMessage(`{"c":3,"d":4}`)
		w.Append("a", 1)
		w.Embed(embedded)
		w.Append("b", 2)
		got, err := w.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"a":1,"c":3,"d":4,"b":2}`
		if string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("optional fields", func(t *testing.T) {
		var w jsonObjectWriter
		w.Append("a", 0) // assess that a zero value is actually added.
		w.Optional("b", "")
		w.Optional("c", 0)
		w.Optional("d", "hello")
		got, err := w.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"a":0,"d":"hello"}`
		if string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("embed from", func(t *testing.T) {
		var w jsonObjectWriter
		embedded := struct {
			C int    `json:"c"`
			D string `json:"d"`
		}{
			C: 3,
			D: "hello",
		}
		w.Append("a", 1)
		w.EmbedFrom(embedded)
		w.Append("b", 2)
		got, err := w.MarshalJSON()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"a":1,"c":3,"d":"hello","b":2}`
		if string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
