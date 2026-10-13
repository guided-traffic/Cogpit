package kinds_test

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzConfigMaps(f *testing.F) { fuzzProfile(f, "configmaps") }

// Config Maps have no status token (inferred: a Config Map has no state).
func TestConfigMapTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"configmaps", "healthy", "keys", "FEATURE_FLAGS, LOG_LEVEL, app.properties", model.StatusNone, model.StatusNone},
		{"configmaps", "failing", "keys", "upstreams.conf, worker_connections", model.StatusNone, model.StatusNone},
		{"configmaps", "terminating", "keys", "holder", model.StatusNone, model.StatusTerminating},
		{"configmaps", "minimal", "keys", "", model.StatusNone, model.StatusNone},
		// binaryData keys are listed with the data keys, sorted together.
		{"configmaps", "binary", "keys", "favicon.ico, theme.css, tiny.bin", model.StatusNone, model.StatusNone},
	})
	assertColumnOrder(t, "configmaps", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "keys", Title: "Keys"},
		{ID: "age", Title: "Age"},
	})
}

// TestConfigMapDrawerCaps builds the largest Config Map the API server accepts in spirit (many
// keys, long values, multi-byte text) and checks that the drawer stays small and cuts cleanly.
func TestConfigMapDrawerCaps(t *testing.T) {
	p, _ := kinds.ByKey("configmaps")
	data := map[string]any{}
	for i := range 150 {
		// 3-byte runes, so that a byte cap lands inside a rune unless the cut respects them.
		data["key-"+strconv.Itoa(1000+i)] = strings.Repeat("€", 3000)
	}
	data["a-many-lines"] = strings.Repeat("line\n", 500)
	binary := map[string]any{}
	for i := range 120 {
		binary["bin-"+strconv.Itoa(1000+i)] = "QUJD"
	}
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "big", "namespace": "default"},
		"data":     data, "binaryData": binary,
	}}

	var dataItems, binItems []model.DetailItem
	for _, s := range p.Detail(obj) {
		switch s.Title {
		case "Data":
			dataItems = s.Items
		case "Binary Data":
			binItems = s.Items
		}
	}
	if len(dataItems) != 101 || dataItems[100].Value != "51 more keys: see the YAML" {
		t.Fatalf("data: %d items, last %+v; want 100 keys and a note on 51 more", len(dataItems), dataItems[len(dataItems)-1])
	}
	if len(binItems) != 101 || binItems[0].Value != "3 B" || binItems[100].Value != "20 more keys: see the YAML" {
		t.Fatalf("binary data: %d items, first %+v, last %+v", len(binItems), binItems[0], binItems[len(binItems)-1])
	}

	first := dataItems[0] // a-many-lines sorts first: cut by lines
	if first.Name != "a-many-lines" || strings.Count(first.Value, "\n") != 40 || !strings.HasSuffix(first.Value, "the full value is in the YAML]") {
		t.Errorf("many lines: got %d lines, ends %q", strings.Count(first.Value, "\n")+1, configMapTail(first.Value))
	}
	var total, lines int
	notShown := 0
	for _, it := range dataItems[:100] {
		if !utf8.ValidString(it.Value) {
			t.Fatalf("%s: cut inside a rune", it.Name)
		}
		if strings.HasPrefix(it.Value, "[not shown: ") {
			notShown++
		}
		total += len(it.Value)
		lines += strings.Count(it.Value, "\n") + 1
	}
	if notShown == 0 {
		t.Error("the total cap never applied")
	}
	// 32 KiB of values plus one note per key; 400 lines plus one note line per cut value.
	if total > 32<<10+100*80 || lines > 400+100 {
		t.Errorf("the drawer holds %d bytes in %d lines; the caps are 32 KiB and 400 lines", total, lines)
	}
	second := dataItems[1].Value // cut by bytes: 4 KiB of 9000
	if !strings.HasSuffix(second, "[cut: 4.0 KiB of 8.8 KiB shown; the full value is in the YAML]") {
		t.Errorf("byte cut: ends %q", configMapTail(second))
	}
}

func configMapTail(s string) string {
	if len(s) > 80 {
		return s[len(s)-80:]
	}
	return s
}
