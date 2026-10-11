package model

import (
	"encoding/json"
	"math"
	"slices"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func sampleObject() *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"name":              "web-0",
			"creationTimestamp": "2026-10-01T12:00:00Z",
			"labels":            map[string]any{"app": "web", "tier": int64(1)},
		},
		"spec": map[string]any{
			"nodeName": "worker-1",
			"replicas": int64(3),
			"paused":   true,
			"args":     []any{"--a", int64(2), "--b"},
			"port":     "http",
			"nullable": nil,
			"containers": []any{
				map[string]any{"name": "app", "image": "registry.example/app:1"},
			},
		},
		"status": map[string]any{
			"eventTime": "2026-10-11T09:00:00.123456Z",
			"bad":       "yesterday",
			"float":     2.0,
			"half":      2.5,
		},
	}}
}

func TestNestedAccessors(t *testing.T) {
	obj := sampleObject()
	if got := NestedString(obj, "spec", "nodeName"); got != "worker-1" {
		t.Errorf("NestedString = %q", got)
	}
	if got := NestedString(obj, "spec", "replicas"); got != "" {
		t.Errorf("NestedString of a number = %q, want empty", got)
	}
	if got := NestedString(obj, "spec", "nodeName", "deeper"); got != "" {
		t.Errorf("NestedString through a string = %q", got)
	}
	if n, ok := NestedInt64(obj, "spec", "replicas"); n != 3 || !ok {
		t.Errorf("NestedInt64 = %d, %v", n, ok)
	}
	if n, ok := NestedInt64(obj, "status", "float"); n != 2 || !ok {
		t.Errorf("NestedInt64 of a whole float = %d, %v", n, ok)
	}
	if _, ok := NestedInt64(obj, "status", "half"); ok {
		t.Error("NestedInt64 accepted 2.5")
	}
	if b, ok := NestedBool(obj, "spec", "paused"); !b || !ok {
		t.Errorf("NestedBool = %v, %v", b, ok)
	}
	if _, ok := NestedBool(obj, "spec", "nodeName"); ok {
		t.Error("NestedBool accepted a string")
	}
	if got := NestedStrings(obj, "spec", "args"); !slices.Equal(got, []string{"--a", "--b"}) {
		t.Errorf("NestedStrings = %q", got)
	}
	if got := NestedStringMap(obj, "metadata", "labels"); len(got) != 1 || got["app"] != "web" {
		t.Errorf("NestedStringMap = %v", got)
	}
	containers := NestedSlice(obj, "spec", "containers")
	if len(containers) != 1 || NestedString(containers[0], "image") != "registry.example/app:1" {
		t.Errorf("NestedSlice element access failed: %v", containers)
	}
	if NestedMap(obj, "spec") == nil || NestedMap(obj, "spec", "nodeName") != nil {
		t.Error("NestedMap")
	}
	if v, ok := NestedValue(obj, "spec", "nullable"); v != nil || !ok {
		t.Errorf("NestedValue of null = %v, %v; want nil, true", v, ok)
	}
	if _, ok := NestedValue(obj, "spec", "missing"); ok {
		t.Error("NestedValue found a missing field")
	}
	if got := NestedIntOrString(obj, "spec", "port"); got != "http" {
		t.Errorf("NestedIntOrString string = %q", got)
	}
	if got := NestedIntOrString(obj, "spec", "replicas"); got != "3" {
		t.Errorf("NestedIntOrString int = %q", got)
	}
	if got := NestedIntOrString(obj, "spec", "paused"); got != "" {
		t.Errorf("NestedIntOrString bool = %q", got)
	}
	want := time.Date(2026, 10, 11, 9, 0, 0, 123456000, time.UTC)
	if got := NestedTime(obj, "status", "eventTime"); !got.Equal(want) {
		t.Errorf("NestedTime of a MicroTime = %v", got)
	}
	if !NestedTime(obj, "status", "bad").IsZero() || !NestedTime(obj, "status", "float").IsZero() {
		t.Error("NestedTime accepted a non-timestamp")
	}
	if got := NestedTime(map[string]any{"t": want}, "t"); !got.Equal(want) {
		t.Error("NestedTime of a time.Time")
	}
}

func TestNestedRoots(t *testing.T) {
	m := map[string]any{"a": map[string]any{"b": "c"}}
	var nilObj *unstructured.Unstructured
	for _, root := range []any{m, &unstructured.Unstructured{Object: m}, unstructured.Unstructured{Object: m}} {
		if NestedString(root, "a", "b") != "c" {
			t.Errorf("root %T not read", root)
		}
	}
	for _, root := range []any{nil, nilObj, "string", 42, []any{m}, map[string]string{"a": "b"}} {
		if v, ok := NestedValue(root, "a"); v != nil || ok {
			t.Errorf("root %T yielded %v", root, v)
		}
	}
	if v, ok := NestedValue(m); !ok || v == nil {
		t.Error("an empty path should return the root itself")
	}
}

func TestToInt64(t *testing.T) {
	tests := []struct {
		in   any
		want int64
		ok   bool
	}{
		{int64(-5), -5, true}, {int(7), 7, true}, {int32(8), 8, true}, {int16(9), 9, true},
		{int8(10), 10, true}, {uint(11), 11, true}, {uint8(12), 12, true}, {uint16(13), 13, true},
		{uint32(14), 14, true}, {uint64(15), 15, true}, {float32(16), 16, true},
		{json.Number("17"), 17, true}, {json.Number("1.5"), 0, false},
		{uint64(math.MaxUint64), 0, false}, {math.NaN(), 0, false}, {math.Inf(1), 0, false},
		{float64(math.MaxInt64), 0, false}, {float64(math.MinInt64), math.MinInt64, true},
		{"3", 0, false}, {nil, 0, false}, {true, 0, false},
	}
	for _, tt := range tests {
		if got, ok := toInt64(tt.in); got != tt.want || ok != tt.ok {
			t.Errorf("toInt64(%#v) = %d, %v; want %d, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestSortedPairsAndAgeCell(t *testing.T) {
	if got := SortedPairs(map[string]string{"b": "2", "a": "1"}); !slices.Equal(got, []string{"a=1", "b=2"}) {
		t.Errorf("SortedPairs = %q", got)
	}
	if got := SortedPairs(nil); len(got) != 0 {
		t.Errorf("SortedPairs(nil) = %q", got)
	}
	ts := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	if c := AgeCell(ts); c.Num != ts.Unix() || c.Text != "" {
		t.Errorf("AgeCell = %+v", c)
	}
	if c := AgeCell(time.Time{}); c != (Cell{}) {
		t.Errorf("AgeCell(zero) = %+v", c)
	}
}
