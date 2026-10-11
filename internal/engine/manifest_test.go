package engine

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseManifests(t *testing.T) {
	objs, err := ParseManifests(`# leading comment
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: a
data:
  n: "1"
---
---
# only a comment
...
---
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: ConfigMap
  metadata: {name: b}
- apiVersion: apps/v1
  kind: Deployment
  metadata: {name: c}
  spec: {replicas: 3}
`)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, o := range objs {
		names = append(names, o.GetKind()+"/"+o.GetName())
	}
	if !slices.Equal(names, []string{"ConfigMap/a", "ConfigMap/b", "Deployment/c"}) {
		t.Errorf("objects %v", names)
	}
	// Integers stay int64, as unstructured content needs them.
	if r, ok, err := unstructured.NestedInt64(objs[2].Object, "spec", "replicas"); !ok || err != nil || r != 3 {
		t.Errorf("replicas %v %v %v", r, ok, err)
	}

	for name, text := range map[string]string{
		"empty":           "",
		"comments only":   "# nothing\n---\n# here\n",
		"scalar":          "hello",
		"sequence":        "- a\n- b\n",
		"null":            "null\n",
		"no apiVersion":   "kind: ConfigMap\n",
		"no kind":         "apiVersion: v1\n",
		"kind not string": "apiVersion: v1\nkind: 3\n",
		"metadata scalar": "apiVersion: v1\nkind: ConfigMap\nmetadata: x\n",
		"name not string": "apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: [a]\n",
		"broken":          "apiVersion: v1\nkind: [ConfigMap\n",
		"tab indent":      "apiVersion: v1\n\tkind: ConfigMap\n",
		"list no items":   "apiVersion: v1\nkind: List\n",
		"list scalar":     "apiVersion: v1\nkind: List\nitems: [1]\n",
		"list in list":    "apiVersion: v1\nkind: List\nitems:\n- apiVersion: v1\n  kind: List\n  items: []\n",
		"too large":       strings.Repeat("#", MaxManifestBytes+1),
	} {
		if _, err := ParseManifests(text); !errors.Is(err, ErrManifest) {
			t.Errorf("%s: %v, want ErrManifest", name, err)
		}
	}
	many := strings.Repeat("apiVersion: v1\nkind: ConfigMap\nmetadata: {name: x}\n---\n", MaxManifestDocuments+1)
	if _, err := ParseManifests(many); !errors.Is(err, ErrManifest) {
		t.Errorf("too many documents: %v", err)
	}
	if _, err := ParseManifest("apiVersion: v1\nkind: A\n---\napiVersion: v1\nkind: B\n"); err == nil || !strings.Contains(err.Error(), "holds 2 objects") {
		t.Errorf("two objects for one edit: %v", err)
	}
}

func TestRedactedKind(t *testing.T) {
	if !RedactedKind("v1", "Secret") {
		t.Error("Secrets are redacted")
	}
	for _, k := range [][2]string{{"v1", "ConfigMap"}, {"example.io/v1", "Secret"}, {"v1", "secret"}} {
		if RedactedKind(k[0], k[1]) {
			t.Errorf("%s %s reported as redacted", k[0], k[1])
		}
	}
}

func TestEditableYAMLDropsManagedFields(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "a", "managedFields": []any{map[string]any{"manager": "kubectl"}}},
	}}
	text, err := EditableYAML(obj)
	if err != nil || strings.Contains(text, "managedFields") || !strings.Contains(text, "name: a") {
		t.Errorf("%q, %v", text, err)
	}
	if _, ok := obj.Object["metadata"].(map[string]any)["managedFields"]; !ok {
		t.Error("EditableYAML changed the cached object")
	}
}

func TestCompactDiff(t *testing.T) {
	before := "a\nb\nc\nd\ne\nf\ng\nh\ni\nj\n"
	after := "a\nb\nc\nd\nE\nf\ng\nh\ni\nj\n"
	lines := DiffLines(before, after)
	got := CompactDiff(lines, 2)
	var ops []string
	for _, l := range got {
		ops = append(ops, string(l.Op)+l.Text)
	}
	if !slices.Equal(ops, []string{"=c", "=d", "-e", "+E", "=f", "=g"}) {
		t.Errorf("compact %v", ops)
	}
	if len(CompactDiff(lines, -1)) != len(lines) {
		t.Error("context -1 must keep every line")
	}
	if d := CompactDiff(DiffLines(before, before), 3); d == nil || len(d) != 0 {
		t.Errorf("an unchanged diff compacts to an empty list, got %v", d)
	}
}

func TestTemplates(t *testing.T) {
	all, err := Templates()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tp := range all {
		names = append(names, tp.Name)
		obj, err := ParseManifest(tp.Text)
		if err != nil {
			t.Errorf("%s: %v", tp.Name, err)
			continue
		}
		if obj.GetNamespace() != "" || obj.GetName() == "" || RedactedKind(obj.GetAPIVersion(), obj.GetKind()) {
			t.Errorf("%s: namespace %q name %q", tp.Name, obj.GetNamespace(), obj.GetName())
		}
	}
	if !slices.IsSorted(names) || names[0] != "ClusterRole" || !slices.Contains(names, DefaultTemplate) {
		t.Errorf("templates %v", names)
	}
	if tp, err := TemplateByName("deployment"); err != nil || tp.Name != "Deployment" || tp.APIVersion != "apps/v1" || !tp.Namespaced {
		t.Errorf("TemplateByName(deployment) = %+v, %v", tp, err)
	}
	if tp, err := TemplateByName("Namespace"); err != nil || tp.Namespaced {
		t.Errorf("Namespace template %+v, %v", tp, err)
	}
}

// FuzzParseManifests: any text yields objects that pass the checks, or an ErrManifest; never a
// panic. It covers what object.apply and object.create parse.
func FuzzParseManifests(f *testing.F) {
	for _, s := range []string{
		"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n",
		"apiVersion: v1\nkind: List\nitems:\n- apiVersion: v1\n  kind: Pod\n  metadata: {name: p}\n",
		"---\n---\n", "a: &x [*x]\n", "a: !!binary aGk=\n", "{\"apiVersion\":\"v1\",\"kind\":\"X\"}",
		"apiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: a\n---\nkind: [\n",
		"a: &a [*a, *a, *a]\nb: &b [*a, *a]\nc: [*b, *b, *b]\n", "apiVersion: 1e400\nkind: x\n",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, text string) {
		objs, err := ParseManifests(text)
		if err != nil {
			if !errors.Is(err, ErrManifest) {
				t.Fatalf("error does not wrap ErrManifest: %v", err)
			}
			return
		}
		for _, o := range objs {
			if o.GetAPIVersion() == "" || o.GetKind() == "" {
				t.Fatalf("object without apiVersion or kind: %v", o.Object)
			}
			if _, err := EditableYAML(o); err != nil {
				t.Fatalf("parsed object not representable: %v", err)
			}
		}
	})
}
