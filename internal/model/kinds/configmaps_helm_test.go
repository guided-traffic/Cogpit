package kinds_test

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// The drawer of a Helm release record kept in a ConfigMap (HELM_DRIVER=configmap) shows its
// payload's size, never the payload, which holds the release's values; the record's other keys,
// and a key named release in a ConfigMap Helm did not write, show as before.
func TestConfigMapDrawerHidesHelmRecords(t *testing.T) {
	p, _ := kinds.ByKey("configmaps")
	payload := strings.Repeat("H4sIAAAAAAAC", 300)
	record := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": "sh.helm.release.v1.web.v1", "namespace": "apps",
			"labels": map[string]any{"owner": "helm", "name": "web", "version": "1"}},
		"data": map[string]any{"release": payload, "extra": "visible"},
	}}
	items := map[string]string{}
	for _, s := range p.Detail(record) {
		for _, it := range s.Items {
			items[it.Name] = it.Value
		}
	}
	if v := items["release"]; strings.Contains(v, "H4sI") || !strings.Contains(v, "Helm release record, 3.5 KiB") {
		t.Errorf("release %q", v)
	}
	if items["extra"] != "visible" {
		t.Errorf("extra %q", items["extra"])
	}
	cells, _ := p.Cells(record)
	if cells[2].Text != "extra, release" {
		t.Errorf("keys %q", cells[2].Text)
	}

	plain := record.DeepCopy()
	plain.SetLabels(map[string]string{"app": "web"})
	for _, s := range p.Detail(plain) {
		for _, it := range s.Items {
			if it.Name == "release" && !strings.HasPrefix(it.Value, "H4sI") {
				t.Errorf("a ConfigMap Helm did not write: %q", it.Value)
			}
		}
	}
	if model.Redact(plain) != plain {
		t.Error("a ConfigMap Helm did not write is redacted")
	}
}
