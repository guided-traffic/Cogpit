package kinds_test

import (
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestWithholdFluxCredentials (H-63): each credential field of each Flux kind is withheld, a
// source URL's user information too (SSH's "git@" stays), and kubectl's last-applied annotation,
// which repeats them; the object passed in is not changed; other kinds and Flux objects without
// such a field come back as they are.
func TestWithholdFluxCredentials(t *testing.T) {
	for _, c := range []struct {
		name      string
		obj       *unstructured.Unstructured
		secret    string
		want      string
		unchanged bool
	}{
		{"helm release values", testkit.Object("helm.toolkit.fluxcd.io/v2", "HelmRelease", "shop", "web", map[string]any{
			"spec": map[string]any{"values": map[string]any{"password": "hr-pass-1"}, "interval": "5m"}}), "hr-pass-1", "values: '" + model.Withheld, false},
		{"helm release v2beta1", testkit.Object("helm.toolkit.fluxcd.io/v2beta1", "HelmRelease", "shop", "web", map[string]any{
			"spec": map[string]any{"values": map[string]any{"token": "hr-pass-2"}}}), "hr-pass-2", "values: '" + model.Withheld, false},
		{"provider address and proxy", testkit.Object("notification.toolkit.fluxcd.io/v1beta3", "Provider", "flux-system", "slack", map[string]any{
			"spec": map[string]any{"address": "https://hooks.slack.com/services/hook-1", "proxy": "http://u:proxy-pass@proxy:3128"}}),
			"hook-1", "proxy: '" + model.Withheld, false},
		{"receiver webhook path", testkit.Object("notification.toolkit.fluxcd.io/v1", "Receiver", "flux-system", "gh", map[string]any{
			"status": map[string]any{"webhookPath": "/hook/receiver-digest-1"}}), "receiver-digest-1", "webhookPath: '" + model.Withheld, false},
		{"git URL with a token", testkit.Object("source.toolkit.fluxcd.io/v1", "GitRepository", "flux-system", "apps", map[string]any{
			"spec": map[string]any{"url": "https://flux:git-token-1@github.com/acme/apps"}}), "git-token-1", "url: https://<withheld>@github.com/acme/apps", false},
		{"helm repository URL", testkit.Object("source.toolkit.fluxcd.io/v1", "HelmRepository", "flux-system", "charts", map[string]any{
			"spec": map[string]any{"url": "https://u:helm-pass-1@charts.example.io"}}), "helm-pass-1", "url: https://<withheld>@charts.example.io", false},
		{"OCI URL", testkit.Object("source.toolkit.fluxcd.io/v1beta2", "OCIRepository", "flux-system", "oci", map[string]any{
			"spec": map[string]any{"url": "oci://robot:oci-pass-1@ghcr.io/acme/manifests"}}), "oci-pass-1", "url: oci://<withheld>@ghcr.io/acme/manifests", false},
		{"bucket endpoint", testkit.Object("source.toolkit.fluxcd.io/v1", "Bucket", "flux-system", "b", map[string]any{
			"spec": map[string]any{"endpoint": "https://key:bucket-secret-1@minio.example.io"}}), "bucket-secret-1", "endpoint: https://<withheld>@minio.example.io", false},
		{"SSH git URL", testkit.Object("source.toolkit.fluxcd.io/v1", "GitRepository", "flux-system", "ssh", map[string]any{
			"spec": map[string]any{"url": "ssh://git@github.com/acme/apps"}}), "", "url: ssh://git@github.com/acme/apps", true},
		{"helm release without values", testkit.Object("helm.toolkit.fluxcd.io/v2", "HelmRelease", "shop", "plain", map[string]any{
			"spec": map[string]any{"valuesFrom": []any{map[string]any{"kind": "Secret", "name": "vals"}}}}), "", "valuesFrom", true},
		{"a ConfigMap", testkit.Object("v1", "ConfigMap", "shop", "values", map[string]any{
			"data": map[string]any{"values": "x"}}), "", "values: x", true},
		{"a look-alike outside Flux", testkit.Object("example.com/v1", "HelmRelease", "shop", "web", map[string]any{
			"spec": map[string]any{"values": "kept-1"}}), "", "values: kept-1", true},
	} {
		before, _ := yaml.Marshal(c.obj.Object)
		out, changed := kinds.WithholdFluxCredentials(c.obj)
		after, _ := yaml.Marshal(c.obj.Object)
		got, _ := yaml.Marshal(out.Object)
		switch {
		case string(before) != string(after):
			t.Errorf("%s: the object passed in changed", c.name)
		case changed == c.unchanged:
			t.Errorf("%s: changed %v", c.name, changed)
		case c.secret != "" && strings.Contains(string(got), c.secret):
			t.Errorf("%s: the credential stays:\n%s", c.name, got)
		case !strings.Contains(string(got), c.want):
			t.Errorf("%s: want %q in:\n%s", c.name, c.want, got)
		}
	}

	applied := testkit.Object("helm.toolkit.fluxcd.io/v2", "HelmRelease", "shop", "web", map[string]any{
		"metadata": map[string]any{"annotations": map[string]any{"kubectl.kubernetes.io/last-applied-configuration": `{"spec":{"values":{"password":"applied-1"}}}`}}})
	out, changed := kinds.WithholdFluxCredentials(applied)
	if raw, _ := yaml.Marshal(out.Object); !changed || strings.Contains(string(raw), "applied-1") {
		t.Errorf("the last-applied annotation: %s", raw)
	}
}
