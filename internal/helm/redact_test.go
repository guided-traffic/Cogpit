package helm

import (
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

// A rendered manifest's Secrets lose their values, in every shape a chart can render them;
// everything else is kept byte for byte, Helm's "# Source:" comments included.
func TestRedactManifest(t *testing.T) {
	manifest := `---
# Source: web/templates/cm.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: web
data:
  note: "the Secret web-db holds the password"
---
# Source: web/templates/secret.yaml
apiVersion: v1
kind: Secret
metadata:
  name: web-db
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: '{"stringData":{"password":"hunter2"}}'
type: Opaque
stringData:
  password: hunter2
data:
  token: aHVudGVyMg==
---
# Source: web/templates/list.yaml
apiVersion: v1
kind: List
items:
- apiVersion: v1
  kind: Secret
  metadata:
    name: in-a-list
  data:
    key: aHVudGVyMg==
- apiVersion: v1
  kind: ConfigMap
  metadata:
    name: kept
  data:
    plain: visible
---
# Source: web/templates/secretlist.yaml
apiVersion: v1
kind: SecretList
items:
- metadata:
    name: no-kind
  stringData:
    pin: hunter2
---
# Source: web/templates/json.yaml
{"apiVersion": "v1", "kind": "Secret", "metadata": {"name": "as-json"}, "data": {"k": "aHVudGVyMg=="}}
---
# Source: web/templates/broken.yaml
apiVersion: v1
kind: Secret
data: [unclosed
  password: hunter2
---
# Source: web/templates/strange.yaml
apiVersion: v1
kind: Secret
metadata:
  name: data-not-a-map
data: hunter2
`
	got := redactManifest(manifest)
	for _, leak := range []string{"hunter2", "aHVudGVyMg=="} {
		if strings.Contains(got, leak) {
			t.Errorf("%q survived:\n%s", leak, got)
		}
	}
	for _, kept := range []string{
		"# Source: web/templates/cm.yaml\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: web\ndata:\n  note: \"the Secret web-db holds the password\"\n",
		"# Source: web/templates/secret.yaml\n", "plain: visible", "name: as-json", "name: data-not-a-map",
		"password: " + model.Redacted, "# Source: web/templates/broken.yaml\n# [",
	} {
		if !strings.Contains(got, kept) {
			t.Errorf("missing %q in:\n%s", kept, got)
		}
	}
	if n := strings.Count(got, "\n---\n") + 1; n != strings.Count(manifest, "\n---\n")+1 {
		t.Errorf("%d documents, want those of the input", n)
	}

	plain := "---\n# Source: x\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: x\n"
	if redactManifest(plain) != plain {
		t.Error("a manifest without a Secret changed")
	}
	mentions := "apiVersion: v1\nkind: ConfigMap\ndata:\n  hint: \"kind: Secret\"\n"
	if redactManifest(mentions) != mentions {
		t.Error("a ConfigMap that mentions a Secret changed")
	}
}

func FuzzRedactManifest(f *testing.F) {
	f.Add("apiVersion: v1\nkind: Secret\ndata:\n  k: dg==\n")
	f.Add("---\nkind: List\nitems:\n- kind: Secret\n  stringData: {a: b}\n")
	f.Add("{\"kind\":\"Secret\",\"data\":\"x\"}")
	f.Fuzz(func(_ *testing.T, s string) {
		_ = redactManifest(s) // untrusted input: it must never panic
	})
}
