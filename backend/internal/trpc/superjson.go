package trpc

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// Date is a time that superjson turns into a JavaScript Date. In JSON it is
// written like Date.prototype.toISOString.
type Date time.Time

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(d).UTC().Format("2006-01-02T15:04:05.000Z"))
}

// MapEntry is one key and value of a Map.
type MapEntry struct {
	Key   any
	Value any
}

// Map is an ordered map that superjson turns into a JavaScript Map. In JSON it
// is written as [[key, value], …].
type Map []MapEntry

func (m Map) MarshalJSON() ([]byte, error) {
	pairs := make([][2]any, len(m))
	for i, e := range m {
		pairs[i] = [2]any{e.Key, e.Value}
	}
	return json.Marshal(pairs)
}

var (
	dateType = reflect.TypeFor[Date]()
	mapType  = reflect.TypeFor[Map]()
)

// meta returns superjson's metadata for v: the paths of its Dates and Maps,
// or nil when it has none.
func meta(v any) map[string]any {
	values := map[string][]string{}
	walk(reflect.ValueOf(v), nil, values)
	if len(values) == 0 {
		return nil
	}
	return map[string]any{"values": values}
}

// walk records the superjson type of every Date and Map in v under its path,
// following the names encoding/json writes.
func walk(v reflect.Value, path []string, values map[string][]string) {
	if !v.IsValid() {
		return
	}
	switch v.Type() {
	case dateType:
		values[joinPath(path)] = []string{"Date"}
		return
	case mapType:
		values[joinPath(path)] = []string{"map"}
		return
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			walk(v.Elem(), path, values)
		}
	case reflect.Struct:
		t := v.Type()
		for i := range t.NumField() {
			f := t.Field(i)
			// encoding/json promotes the fields of embedded structs, also
			// of unexported types.
			if !f.IsExported() && !f.Anonymous {
				continue
			}
			name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "-" && opts == "" {
				continue
			}
			fv := v.Field(i)
			if strings.Contains(opts, "omitempty") && fv.IsZero() {
				continue
			}
			if f.Anonymous && name == "" {
				walk(fv, path, values)
				continue
			}
			if name == "" {
				name = f.Name
			}
			walk(fv, append(path, name), values)
		}
	case reflect.Slice, reflect.Array:
		for i := range v.Len() {
			walk(v.Index(i), append(path, strconv.Itoa(i)), values)
		}
	case reflect.Map:
		if v.Type().Key().Kind() != reflect.String {
			return
		}
		iter := v.MapRange()
		for iter.Next() {
			walk(iter.Value(), append(path, iter.Key().String()), values)
		}
	}
}

// joinPath joins a path as superjson does, escaping backslashes and dots.
func joinPath(path []string) string {
	escaped := make([]string, len(path))
	for i, p := range path {
		escaped[i] = strings.ReplaceAll(strings.ReplaceAll(p, `\`, `\\`), ".", `\.`)
	}
	return strings.Join(escaped, ".")
}
