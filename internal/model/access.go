package model

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// The Nested* accessors read unstructured content without copying it and without ever panicking.
// Their root may be an object (*unstructured.Unstructured or unstructured.Unstructured), a plain
// map[string]any, or an element taken from a slice another accessor returned (a container, a
// condition). Anything else, a missing field, a null or a value of the wrong type yields the zero
// value, so callers never type-assert unstructured data themselves.
//
// Unlike unstructured.NestedSlice and friends they do not deep-copy: deep copies cost
// allocations on every watch event, and DeepCopyJSONValue panics on Go types JSON never yields.
// The results alias the object, so callers must not modify them.

// NestedValue returns the value at the path of field names, and whether it exists. A null value
// exists and is nil.
func NestedValue(root any, fields ...string) (any, bool) {
	m := rootMap(root)
	if m == nil {
		return nil, false
	}
	var cur any = m
	for _, f := range fields {
		cm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		if cur, ok = cm[f]; !ok {
			return nil, false
		}
	}
	return cur, true
}

// NestedString returns the string at the path, or "" when it is missing or not a string.
func NestedString(root any, fields ...string) string {
	v, _ := NestedValue(root, fields...)
	s, _ := v.(string)
	return s
}

// NestedInt64 returns the integer at the path and whether there is one. JSON decoders yield
// int64 or float64, Go-built objects any integer type; a float counts only when it is a whole
// number in range, so 1.5, NaN and Inf are not integers.
func NestedInt64(root any, fields ...string) (int64, bool) {
	v, _ := NestedValue(root, fields...)
	return toInt64(v)
}

// NestedBool returns the boolean at the path and whether there is one.
func NestedBool(root any, fields ...string) (value, ok bool) {
	v, _ := NestedValue(root, fields...)
	value, ok = v.(bool)
	return value, ok
}

// NestedSlice returns the list at the path, or nil. The elements are untyped: read them with
// the Nested* accessors.
func NestedSlice(root any, fields ...string) []any {
	v, _ := NestedValue(root, fields...)
	s, _ := v.([]any)
	return s
}

// NestedMap returns the object at the path, or nil.
func NestedMap(root any, fields ...string) map[string]any {
	v, _ := NestedValue(root, fields...)
	m, _ := v.(map[string]any)
	return m
}

// NestedStrings returns the strings of the list at the path, skipping elements that are not
// strings. It returns nil when the list is missing.
func NestedStrings(root any, fields ...string) []string {
	s := NestedSlice(root, fields...)
	if len(s) == 0 {
		return nil
	}
	out := make([]string, 0, len(s))
	for _, e := range s {
		if str, ok := e.(string); ok {
			out = append(out, str)
		}
	}
	return out
}

// NestedStringMap returns the string entries of the object at the path (labels, a selector),
// skipping entries whose value is not a string. It returns nil when the object is missing.
func NestedStringMap(root any, fields ...string) map[string]string {
	m := NestedMap(root, fields...)
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}
	return out
}

// NestedTime returns the timestamp at the path, or the zero time when it is missing or not an
// RFC 3339 string. It reads metav1.Time and metav1.MicroTime fields alike: Go's RFC 3339 parser
// accepts the fractional seconds MicroTime carries.
func NestedTime(root any, fields ...string) time.Time {
	v, _ := NestedValue(root, fields...)
	switch t := v.(type) {
	case string:
		if t == "" {
			return time.Time{}
		}
		parsed, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return time.Time{}
		}
		return parsed
	case time.Time:
		return t
	}
	return time.Time{}
}

// NestedIntOrString returns an int-or-string field (a target port, maxSurge) as text: the string
// itself, or the integer in decimal. It returns "" when the field is missing or of another type.
func NestedIntOrString(root any, fields ...string) string {
	v, _ := NestedValue(root, fields...)
	if s, ok := v.(string); ok {
		return s
	}
	if n, ok := toInt64(v); ok {
		return strconv.FormatInt(n, 10)
	}
	return ""
}

// SortedPairs returns a string map as "key=value" pairs, sorted. Labels, annotations and
// selectors render this way.
func SortedPairs(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k, v := range m {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

// AgeCell returns the cell of an age column for a timestamp: Num is the Unix time in seconds and
// [CellText] formats it relative to now. The zero time gives an empty cell.
func AgeCell(t time.Time) Cell {
	if t.IsZero() {
		return Cell{}
	}
	return Cell{Num: t.Unix()}
}

func rootMap(root any) map[string]any {
	switch r := root.(type) {
	case map[string]any:
		return r
	case *unstructured.Unstructured:
		if r == nil {
			return nil
		}
		return r.Object
	case unstructured.Unstructured:
		return r.Object
	}
	return nil
}

// toInt64 converts the numeric types unstructured content holds. Unsigned values above
// math.MaxInt64 and floats that are not whole numbers in range are rejected rather than wrapped.
func toInt64(v any) (int64, bool) {
	switch n := v.(type) {
	case int64:
		return n, true
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int16:
		return int64(n), true
	case int8:
		return int64(n), true
	case uint64:
		if n > math.MaxInt64 {
			return 0, false
		}
		return int64(n), true
	case uint:
		if uint64(n) > math.MaxInt64 {
			return 0, false
		}
		return int64(n), true
	case uint32:
		return int64(n), true
	case uint16:
		return int64(n), true
	case uint8:
		return int64(n), true
	case float64:
		return floatToInt64(n)
	case float32:
		return floatToInt64(float64(n))
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return i, true
	}
	return 0, false
}

func floatToInt64(f float64) (int64, bool) {
	// -2^63 converts exactly; 2^63 does not fit, hence the asymmetric bounds.
	if math.IsNaN(f) || f != math.Trunc(f) || f < math.MinInt64 || f >= math.MaxInt64 {
		return 0, false
	}
	return int64(f), true
}
