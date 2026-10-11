package model

import (
	"flag"
	"slices"
	"sort"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// The kinds goldens regenerate with `go test ./internal/model/... -update`, which hands the flag
// to this package too; model has no golden files, so it only accepts the flag.
var _ = flag.Bool("update", false, "rewrite golden files (used by package kinds)")

var now = time.Date(2026, 10, 11, 12, 0, 0, 0, time.UTC)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{-time.Hour, "0s"},
		{0, "0s"},
		{59 * time.Second, "59s"},
		{time.Minute, "1m"},
		{59*time.Minute + 59*time.Second, "59m"},
		{time.Hour, "1h"},
		{23 * time.Hour, "23h"},
		{24 * time.Hour, "1d"},
		{364 * 24 * time.Hour, "364d"},
		{365 * 24 * time.Hour, "1y"},
		{3 * 365 * 24 * time.Hour, "3y"},
	}
	for _, tt := range tests {
		if got := FormatAge(tt.d); got != tt.want {
			t.Errorf("FormatAge(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestCellText(t *testing.T) {
	age := Column{ID: "age", Type: ColumnAge}
	num := Column{ID: "restarts", Type: ColumnNumber}
	text := Column{ID: "name", Type: ColumnText}
	tests := []struct {
		col  Column
		cell Cell
		want string
	}{
		{age, Cell{Num: now.Add(-90 * time.Minute).Unix()}, "1h"},
		{age, Cell{Text: "-"}, "-"},
		{age, Cell{}, ""},
		{num, Cell{Num: 42}, "42"},
		{num, Cell{Num: 0}, "0"},
		{num, Cell{Text: "3/4", Num: 3}, "3/4"},
		{text, Cell{Text: "web", Num: 9}, "web"},
	}
	for _, tt := range tests {
		if got := CellText(tt.col, tt.cell, now); got != tt.want {
			t.Errorf("CellText(%s, %+v) = %q, want %q", tt.col.Type, tt.cell, got, tt.want)
		}
	}
}

func TestMatchesFilter(t *testing.T) {
	row := Row{Cells: []Cell{{Text: "web-0"}, {Text: "Shop"}, {Text: "CrashLoopBackOff"}}}
	tests := []struct {
		filter string
		want   bool
	}{
		{"", true},
		{"   ", true},
		{"web", true},
		{"SHOP crash", true},
		{"web missing", false},
		{"loopback", true},
		{"shop-web", false},
	}
	for _, tt := range tests {
		if got := MatchesFilter(row, tt.filter); got != tt.want {
			t.Errorf("MatchesFilter(%q) = %v, want %v", tt.filter, got, tt.want)
		}
	}
}

func TestObjectRef(t *testing.T) {
	pod := ObjectRef{Context: "kind-a", Kind: "pods", Namespace: "shop", Name: "web-0"}
	node := ObjectRef{Kind: "nodes", Name: "worker-1"}
	if pod.Key() != "shop/web-0" || node.Key() != "worker-1" {
		t.Errorf("Key = %q, %q", pod.Key(), node.Key())
	}
	if pod.String() != "pods/shop/web-0" || node.String() != "nodes/worker-1" {
		t.Errorf("String = %q, %q", pod.String(), node.String())
	}
	anyContext := pod
	anyContext.Context = ""
	other := pod
	other.Context = "kind-b"
	renamed := pod
	renamed.Name = "web-1"
	switch {
	case !pod.Same(pod), !pod.Same(anyContext), !anyContext.Same(pod):
		t.Error("Same should match itself and an empty context")
	case pod.Same(other), pod.Same(renamed), pod.Same(node):
		t.Error("Same matched a different object")
	}
}

func TestRedact(t *testing.T) {
	secret := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Secret",
		"metadata": map[string]any{"name": "db", "annotations": map[string]any{
			"kubectl.kubernetes.io/last-applied-configuration": `{"data":{"password":"aHVudGVyMg=="}}`,
			"owner": "team-a",
		}},
		"data":       map[string]any{"password": "aHVudGVyMg=="},
		"stringData": map[string]any{"user": "admin"},
		"type":       "Opaque",
	}}
	out := Redact(secret)
	if out == secret {
		t.Fatal("Redact must copy a Secret")
	}
	if NestedString(out, "data", "password") != Redacted || NestedString(out, "stringData", "user") != Redacted {
		t.Errorf("values not redacted: %v", out.Object)
	}
	ann := out.GetAnnotations()
	if ann["kubectl.kubernetes.io/last-applied-configuration"] != Redacted || ann["owner"] != "team-a" {
		t.Errorf("annotations = %v", ann)
	}
	if NestedString(secret, "data", "password") != "aHVudGVyMg==" {
		t.Error("Redact modified its input")
	}
	cm := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "ConfigMap"}}
	if Redact(cm) != cm || Redact(nil) != nil {
		t.Error("non-Secrets are returned unchanged")
	}
	odd := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Secret", "data": "garbage"}}
	if Redact(odd) == nil {
		t.Error("a Secret with malformed data should still be returned")
	}
}

func TestMetadataSection(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"name":              "web-0",
			"namespace":         "shop",
			"uid":               "uid-1",
			"creationTimestamp": "2026-10-01T12:00:00Z",
			"labels":            map[string]any{"b": "2", "a": "1"},
			"annotations": map[string]any{
				"kubectl.kubernetes.io/last-applied-configuration": "{}",
				"note": "x",
			},
			"ownerReferences": []any{map[string]any{"kind": "ReplicaSet", "name": "web-1a2b"}},
		},
	}}
	s := MetadataSection(obj, now)
	got := map[string]string{}
	var order []string
	for _, it := range s.Items {
		got[it.Name] = it.Value
		order = append(order, it.Name)
	}
	want := map[string]string{
		"Created":       "10d ago (2026-10-01T12:00:00Z)",
		"Name":          "web-0",
		"Namespace":     "shop",
		"Labels":        "a=1\nb=2",
		"Annotations":   "note=x",
		"Controlled By": "ReplicaSet web-1a2b",
		"UID":           "uid-1",
	}
	if s.Title != "Metadata" || len(got) != len(want) {
		t.Fatalf("section = %+v", s)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if order[0] != "Created" || order[len(order)-1] != "UID" {
		t.Errorf("order = %v", order)
	}

	bare := MetadataSection(&unstructured.Unstructured{Object: map[string]any{"metadata": map[string]any{"name": "n"}}}, now)
	if len(bare.Items) != 1 || bare.Items[0].Name != "Name" {
		t.Errorf("bare metadata = %+v", bare.Items)
	}
}

func testProfile() *KindProfile {
	return &KindProfile{
		Key: "widgets", Group: "example.com", Version: "v1", Resource: "widgets", Kind: "Widget",
		Columns: []Column{{ID: "name"}, {ID: "size", Type: ColumnNumber}, {ID: "age", Type: ColumnAge}},
		Cells: func(obj *unstructured.Unstructured) ([]Cell, Status) {
			n, _ := NestedInt64(obj, "spec", "size")
			return []Cell{{Text: obj.GetName()}, {Num: n}}, StatusOK // one cell short on purpose
		},
		Fields: map[string]func(*unstructured.Unstructured) []string{
			"color": func(obj *unstructured.Unstructured) []string {
				return []string{NestedString(obj, "spec", "color")}
			},
			"boom": func(*unstructured.Unstructured) []string { panic("boom") },
		},
	}
}

func widget(name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]any{
		"metadata": map[string]any{
			"name": name, "namespace": "ns", "uid": "u1", "resourceVersion": "7",
			"labels": map[string]any{"a": "1", "b": "2"},
		},
		"spec": map[string]any{"size": int64(5), "color": "blue"},
	}}
}

func TestProfileBasics(t *testing.T) {
	p := testProfile()
	if gvr := p.GVR(); gvr.Group != "example.com" || gvr.Version != "v1" || gvr.Resource != "widgets" {
		t.Errorf("GVR = %v", gvr)
	}
	if p.ColumnIndex("size") != 1 || p.ColumnIndex("missing") != -1 {
		t.Error("ColumnIndex")
	}
	names := p.FieldNames()
	if !slices.Equal(names[:3], []string{"name", "namespace", "label"}) || len(names) != 5 {
		t.Errorf("FieldNames = %v", names)
	}
}

func TestBuildRow(t *testing.T) {
	p := testProfile()
	row := BuildRow(p, "ctx", widget("w1"))
	wantRef := ObjectRef{Context: "ctx", Kind: "widgets", Namespace: "ns", Name: "w1"}
	if row.Ref != wantRef || row.UID != "u1" || row.ResourceVersion != "7" || row.Status != StatusOK {
		t.Errorf("row = %+v", row)
	}
	if len(row.Cells) != 3 || row.Cells[1].Num != 5 || row.Cells[2] != (Cell{}) {
		t.Errorf("short Cells not padded: %+v", row.Cells)
	}

	deleted := widget("w2")
	_ = unstructured.SetNestedField(deleted.Object, "2026-10-11T11:00:00Z", "metadata", "deletionTimestamp")
	if BuildRow(p, "ctx", deleted).Status != StatusTerminating {
		t.Error("a deletion timestamp should make the row terminating")
	}

	p.Cells = func(*unstructured.Unstructured) ([]Cell, Status) { panic("bad profile") }
	row = BuildRow(p, "ctx", widget("w3"))
	if row.Status != StatusUnknown || len(row.Cells) != 3 || row.Cells[0].Text != "w3" {
		t.Errorf("panicking Cells = %+v", row)
	}

	p.Cells = nil
	row = BuildRow(p, "ctx", widget("w4"))
	if len(row.Cells) != 3 || row.Cells[0].Text != "w4" || row.Status != StatusNone {
		t.Errorf("nil Cells = %+v", row)
	}

	p.Columns = nil
	p.Cells = func(*unstructured.Unstructured) ([]Cell, Status) { panic("no columns") }
	if row := BuildRow(p, "ctx", widget("w5")); len(row.Cells) != 0 {
		t.Errorf("no columns = %+v", row.Cells)
	}
}

func TestFieldValues(t *testing.T) {
	p := testProfile()
	obj := widget("w1")
	labels := FieldValues(p, "label", obj)
	sort.Strings(labels)
	if !slices.Equal(labels, []string{"a=1", "b=2"}) {
		t.Errorf("label = %v", labels)
	}
	if got := FieldValues(p, "labels", obj); len(got) != 2 {
		t.Errorf("labels alias = %v", got)
	}
	if got := FieldValues(p, "name", obj); !slices.Equal(got, []string{"w1"}) {
		t.Errorf("name = %v", got)
	}
	if got := FieldValues(p, "namespace", obj); !slices.Equal(got, []string{"ns"}) {
		t.Errorf("namespace = %v", got)
	}
	if got := FieldValues(p, "color", obj); !slices.Equal(got, []string{"blue"}) {
		t.Errorf("color = %v", got)
	}
	if got := FieldValues(p, "boom", obj); got != nil {
		t.Errorf("a panicking field should yield nil, got %v", got)
	}
	if got := FieldValues(p, "unknown", obj); got != nil {
		t.Errorf("unknown field = %v", got)
	}
}
