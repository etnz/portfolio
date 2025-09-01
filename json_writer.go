// In a new file portfolio/json_writer.go
package portfolio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

// jsonObjectWriter helps construct a JSON object with a specific field order.
// Its zero value is ready to use.
type jsonObjectWriter struct {
	bytes.Buffer
	err error
}

// Embed appends the fields from a raw JSON object (provided as a byte slice)
// into the current JSON object being built. It strips the outer braces of the
// embedded JSON, effectively merging its contents.
func (w *jsonObjectWriter) Embed(rawJSON []byte) *jsonObjectWriter {
	if w.err != nil {
		return w
	}
	trimmed := bytes.TrimSpace(rawJSON)
	if len(trimmed) > 2 && trimmed[0] == '{' && trimmed[len(trimmed)-1] == '}' {
		trimmed = trimmed[1 : len(trimmed)-1]
	}
	if len(trimmed) > 0 {
		w.Write(trimmed)
		w.WriteString(",")
	}
	return w
}

// EmbedFrom marshals the given Go value into a JSON object and then embeds
// its fields into the current JSON object being built. This is useful for
// including nested structures.
func (w *jsonObjectWriter) EmbedFrom(v any) *jsonObjectWriter {
	if w.err != nil {
		return w
	}
	rawJSON, err := json.Marshal(v)
	if err != nil {
		w.err = fmt.Errorf("failed to marshal for embedding: %w", err)
		return w
	}
	return w.Embed(rawJSON)
}

// Append adds a new key-value pair to the JSON object. The value is marshaled
// to JSON using `json.Marshal`.
func (w *jsonObjectWriter) Append(key string, value interface{}) *jsonObjectWriter {
	if w.err != nil {
		return w
	}

	valBytes, err := json.Marshal(value)
	if err != nil {
		w.err = fmt.Errorf("failed to marshal value for key %q: %w", key, err)
		return w
	}

	w.WriteString(fmt.Sprintf("%q:", key))
	w.Write(valBytes)
	w.WriteString(",")
	return w
}

// Optional appends a key-value pair to the JSON object only if the provided
// value is not its type's zero value. This helps in omitting empty or default
// fields from the JSON output.
func (w *jsonObjectWriter) Optional(key string, value interface{}) *jsonObjectWriter {
	if w.err != nil {
		return w
	}
	// Check for zero values
	v := reflect.ValueOf(value)
	if !v.IsValid() || v.IsZero() {
		return w
	}
	return w.Append(key, value)
}

// MarshalJSON finalizes the JSON object construction, wraps the content in
// braces, and returns the complete JSON byte slice. It satisfies the
// `json.Marshaler` interface.
func (w *jsonObjectWriter) MarshalJSON() ([]byte, error) {
	if w.err != nil {
		return nil, w.err
	}

	content := bytes.TrimSuffix(w.Bytes(), []byte(","))
	final := make([]byte, 0, len(content)+2)
	final = append(final, '{')
	final = append(final, content...)
	final = append(final, '}')

	return final, nil
}
