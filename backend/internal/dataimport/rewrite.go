package dataimport

import (
	"bytes"
	"encoding/json"
)

// rewriteIDs returns raw with every string value that is a key of ids
// replaced by its value, in objects and arrays at any depth. Object keys,
// numbers and other strings are kept; numbers keep their exact text.
func rewriteIDs(raw json.RawMessage, ids map[string]string) (json.RawMessage, error) {
	if len(raw) == 0 {
		return raw, nil
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	changed := false
	var walk func(v any) any
	walk = func(v any) any {
		switch t := v.(type) {
		case string:
			if id, ok := ids[t]; ok {
				changed = true
				return id
			}
			return t
		case []any:
			for i := range t {
				t[i] = walk(t[i])
			}
			return t
		case map[string]any:
			for k := range t {
				t[k] = walk(t[k])
			}
			return t
		}
		return v
	}
	v = walk(v)
	if !changed {
		return raw, nil
	}
	return json.Marshal(v)
}

// stringsIn calls fn for every string value in raw, at any depth.
func stringsIn(raw json.RawMessage, fn func(string)) {
	var v any
	if json.Unmarshal(raw, &v) != nil {
		return
	}
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case string:
			fn(t)
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			for _, e := range t {
				walk(e)
			}
		}
	}
	walk(v)
}
