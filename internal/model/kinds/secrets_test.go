package kinds_test

import (
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzSecrets(f *testing.F) { fuzzProfile(f, "secrets") }

// Secrets come from a metadata-only watch (D-13): Keys and Type are empty for what the cache holds,
// filled only from a full Secret, which the cache never holds; no status token (inferred).
func TestSecretTexts(t *testing.T) {
	checkStatusTexts(t, []statusText{
		{"secrets", "healthy", "labels", "app.kubernetes.io/component=database, app.kubernetes.io/name=checkout-api", model.StatusNone, model.StatusNone},
		{"secrets", "healthy", "keys", "", model.StatusNone, model.StatusNone},
		{"secrets", "healthy", "type", "", model.StatusNone, model.StatusNone},
		{"secrets", "failing", "labels", "app=edge-proxy", model.StatusNone, model.StatusNone},
		{"secrets", "terminating", "labels", "rotated=true", model.StatusNone, model.StatusTerminating},
		{"secrets", "minimal", "labels", "", model.StatusNone, model.StatusNone},
		{"secrets", "full-object", "keys", "ca.crt, tls.crt, tls.key", model.StatusNone, model.StatusNone},
		{"secrets", "full-object", "type", "kubernetes.io/tls", model.StatusNone, model.StatusNone},
		{"secrets", "helm-release", "keys", "", model.StatusNone, model.StatusNone},
	})
	assertColumnOrder(t, "secrets", []model.Column{
		{ID: "name", Title: "Name"},
		{ID: "namespace", Title: "Namespace"},
		{ID: "labels", Title: "Labels"},
		{ID: "keys", Title: "Keys"},
		{ID: "type", Title: "Type"},
		{ID: "age", Title: "Age"},
	})
	p, _ := kinds.ByKey("secrets")
	if !p.MetadataOnly || p.LabelSelector != "" || !p.APIObject() {
		t.Errorf("the Secrets view must watch every Secret as metadata only: %+v", p)
	}
}

// The profile never shows a value, not even from a full Secret: its drawer has the Type only.
func TestSecretDetailShowsNoValue(t *testing.T) {
	p, _ := kinds.ByKey("secrets")
	for _, c := range loadCases(t, "secrets") {
		for _, sec := range p.Detail(c.obj) {
			for _, it := range sec.Items {
				if it.Name != "Type" || strings.Contains(it.Value, "LS0t") || strings.Contains(it.Value, "certificate authority") {
					t.Errorf("%s: drawer item %+v", c.name, it)
				}
			}
		}
	}
}
