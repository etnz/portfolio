// In a new file portfolio/json_writer.go
package portfolio

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"unicode"
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

// PrefixFrom marshals the given Go value into a JSON object and then embeds
// its fields into the current JSON object being built, the fields are prefixed with 'prefix' and camelCasing. This is useful for
// including nested structures.
// Currently only supports camelCase
func (w *jsonObjectWriter) PrefixFrom(prefix string, v any) *jsonObjectWriter {
	if w.err != nil {
		return w
	}
	rawJSON, err := json.Marshal(v)
	if err != nil {
		w.err = fmt.Errorf("failed to marshal for embedding: %w", err)
		return w
	}
	// I need to parse the raw json again
	dec := json.NewDecoder(bytes.NewReader(rawJSON))
	dec.UseNumber()
	out := &bytes.Buffer{}
	type frame struct {
		index int
		mod   int
	}
	// Empty stack of frames
	frames := make([]frame, 0, 100)
	var f frame
	printSeparator := func() {
		if f.index > 0 {
			switch f.index % f.mod {
			case 0:
				out.WriteString(",")
			case 1:
				out.WriteString(":")
			}
			f.index++
		} else {
			f.index = 1
		}
	}

	for {
		token, err := dec.Token()
		if err != nil {
			if err != io.EOF {
				w.err = fmt.Errorf("failed to marshal for embedding: %w", err)
			}
			break
		}

		// Parse token
		// delims creates a little state machine
		// after a [
		//      - the next token must be printed without separator,
		//      - next tokens must be printed with a prefix ","
		// after a {
		//      - the next token must be printed without separator,
		//      - next tokens must be printed with a prefix ":"
		//      - next tokens must be printed with a prefix ","
		// after a ]} we close the current frame.
		//
		// Therefore a frame is a token index and a mod
		// 		- starting with -1
		//      - increased at each tokens
		//      - mod is either 1 after a [ or 2 after a {
		// if index is positive, print "," for index%mod==0; print ":" for index%mod==1
		// increase index after each token
		// when ] or } current frame is poped from the frames stack
		// when { or [, current frame is pushed into the frames stack, and a new one is created

		switch t := token.(type) {
		case json.Delim:
			out.WriteRune(rune(t))
			switch t {
			case '}', ']':
				f, frames = frames[len(frames)-1], frames[:len(frames)-1]
			case '[':
				f, frames = frame{index: -1, mod: 1}, append(frames, f)
			case '{':
				f, frames = frame{index: -1, mod: 2}, append(frames, f)
			}
		case bool:
			printSeparator()
			if t {
				out.WriteString("true")
			} else {
				out.WriteString("false")
			}
		case json.Number:
			printSeparator()
			out.WriteString(t.String())
		case string:
			printSeparator()
			// a property is a string, in a {} frame, when next separator is ':'
			if f.index%f.mod == 1 {

				runes := []rune(t)
				runes[0] = unicode.ToUpper(runes[0])
				t = prefix + string(runes)
			}
			b, _ := json.Marshal(t)
			out.Write(b)
		case nil:
			printSeparator()
			out.WriteString("null")
		}

	}

	transformedJSON := out.Bytes()
	// Embed the recomputed rawJSON instead
	return w.Embed(transformedJSON)
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
