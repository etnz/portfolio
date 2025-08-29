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

// Embed appends the fields from a marshaled JSON object, stripping its outer braces.
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

// EmbedFrom marshals the given value and embeds the resulting JSON object.
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

// Append adds a key-value pair, using json.Marshal for the value.
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

// Optional appends a key-value pair if the value is not the zero value of its type.
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

// MarshalJSON finalizes and returns the JSON byte slice, satisfying the json.Marshaler interface.
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
