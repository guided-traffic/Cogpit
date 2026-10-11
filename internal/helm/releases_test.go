package helm

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	chart "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/release/common"
	rspb "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/guided-traffic/cogpit/internal/model"
)

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

// testRelease is one revision of a release, as a test writes it.
type testRelease struct {
	ns, name string
	rev      int
	status   common.Status
	values   map[string]any
	manifest string
}

// secretFor encodes a revision the way Helm's Secrets driver stores it: the Secret's name and
// labels, and data.release as Kubernetes base64 of Helm's own base64 of gzipped JSON.
func secretFor(t testing.TB, r testRelease) *unstructured.Unstructured {
	t.Helper()
	rls := &rspb.Release{
		Name: r.name, Namespace: r.ns, Version: r.rev, Config: r.values, Manifest: r.manifest,
		Info: &rspb.Info{
			Status: r.status, FirstDeployed: t0, LastDeployed: t0.Add(time.Duration(r.rev) * time.Hour),
			Description: fmt.Sprintf("Revision %d", r.rev),
		},
		Chart: &chart.Chart{Metadata: &chart.Metadata{Name: "web", Version: fmt.Sprintf("1.%d.0", r.rev), AppVersion: "2.0"}},
	}
	tt, ok := t.(*testing.T)
	var payload string
	if ok {
		payload = encode(tt, rls, true)
	} else {
		b, _ := json.Marshal(rls)
		payload = base64.StdEncoding.EncodeToString(b)
	}
	name := fmt.Sprintf("sh.helm.release.v1.%s.v%d", r.name, r.rev)
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Secret", "type": "helm.sh/release.v1",
		"metadata": map[string]any{
			"name": name, "namespace": r.ns, "uid": "uid-" + r.ns + "-" + name, "resourceVersion": "1",
			"creationTimestamp": t0.Add(time.Duration(r.rev) * time.Hour).Format(time.RFC3339),
			"labels":            map[string]any{"owner": "helm", "name": r.name, "version": fmt.Sprint(r.rev), "status": string(r.status)},
		},
		"data": map[string]any{"release": base64.StdEncoding.EncodeToString([]byte(payload))},
	}}
}

func newTestMemo() *memo {
	return &memo{cur: map[types.UID]*revision{}, prev: map[types.UID]*revision{}}
}

// releaseSecrets: web with three revisions in apps, web with one in other, db with one in apps.
func releaseSecrets(t *testing.T) []*unstructured.Unstructured {
	return []*unstructured.Unstructured{
		secretFor(t, testRelease{ns: "apps", name: "web", rev: 1, status: common.StatusSuperseded}),
		secretFor(t, testRelease{ns: "apps", name: "web", rev: 3, status: common.StatusDeployed,
			values: map[string]any{"password": "hunter2", "replicas": 2}, manifest: "---\n# Source: web/templates/cm.yaml\napiVersion: v1\nkind: ConfigMap\nmetadata:\n  name: web\n"}),
		secretFor(t, testRelease{ns: "apps", name: "web", rev: 2, status: common.StatusSuperseded}),
		secretFor(t, testRelease{ns: "other", name: "web", rev: 1, status: common.StatusFailed}),
		secretFor(t, testRelease{ns: "apps", name: "db", rev: 1, status: common.StatusPendingInstall}),
	}
}

func byKey(objs []*unstructured.Unstructured) map[string]*unstructured.Unstructured {
	out := map[string]*unstructured.Unstructured{}
	for _, o := range objs {
		out[o.GetNamespace()+"/"+o.GetName()] = o
	}
	return out
}

// Derive groups the release Secrets per namespace and release name, the newest revision winning:
// its summary, its Secret's UID and resourceVersion, and the history newest first.
func TestDeriveGroupsReleasesNewestWins(t *testing.T) {
	p := NewReleaseProfile()
	secrets := releaseSecrets(t)
	objs := p.Derive(secrets)
	if len(objs) != 3 {
		t.Fatalf("%d releases, want 3", len(objs))
	}
	got := byKey(objs)
	web := got["apps/web"]
	if web == nil || got["other/web"] == nil || got["apps/db"] == nil {
		t.Fatalf("releases %v", got)
	}
	if rev, _ := model.NestedInt64(web, "revision"); rev != 3 || web.GetUID() != "uid-apps-sh.helm.release.v1.web.v3" || web.GetResourceVersion() != "1" {
		t.Errorf("web: revision %d uid %s rv %s", rev, web.GetUID(), web.GetResourceVersion())
	}
	row := model.BuildRow(p, "test", web)
	texts := make([]string, len(row.Cells))
	for i, c := range row.Cells {
		texts[i] = model.CellText(p.Columns[i], c, t0.Add(5*time.Hour))
	}
	if strings.Join(texts, "|") != "web|apps|web|3|1.3.0|2.0|deployed|2h" || row.Status != model.StatusOK {
		t.Errorf("row %q status %q", texts, row.Status)
	}
	if model.BuildRow(p, "test", got["apps/db"]).Status != model.StatusPending ||
		model.BuildRow(p, "test", got["other/web"]).Status != model.StatusError {
		t.Error("status tokens")
	}
	history := model.NestedSlice(web, "history")
	if len(history) != 3 || model.NestedString(history[0], "status") != "deployed" || model.NestedString(history[2], "chart") != "web-1.1.0" {
		t.Errorf("history %v", history)
	}
	if b, _ := json.Marshal(objs); strings.Contains(string(b), "hunter2") || strings.Contains(string(b), "ConfigMap") {
		t.Errorf("a release object carries values or the manifest: %s", b)
	}
	if !web.GetCreationTimestamp().Time.Equal(t0) {
		t.Errorf("created %v, want the first deployment", web.GetCreationTimestamp())
	}
}

// The drawer: Properties with the summary and a link to the release Secret, the history as a
// table, the newest revision's values (private) and its manifest.
func TestReleaseDetail(t *testing.T) {
	p := NewReleaseProfile()
	web := byKey(p.Derive(releaseSecrets(t)))["apps/web"]
	secs := p.Detail(web)
	titles := make([]string, len(secs))
	for i, s := range secs {
		titles[i] = s.Title
	}
	if strings.Join(titles, ",") != "Properties,History,Values,Manifest" {
		t.Fatalf("sections %v", titles)
	}
	props := map[string]model.DetailItem{}
	for _, it := range secs[0].Items {
		props[it.Name] = it
	}
	if props["Chart"].Value != "web-1.3.0" || props["Status"].Status != model.StatusOK || props["Revision"].Value != "3" ||
		props["Release Secret"].Link == nil || props["Release Secret"].Link.Kind != "secrets" {
		t.Errorf("properties %+v", props)
	}
	if tb := secs[1].Table; tb == nil || len(tb.Rows) != 3 || tb.Rows[0][0] != "3" || tb.Rows[2][3] != "web-1.1.0" {
		t.Errorf("history %+v", secs[1].Table)
	}
	values := secs[2].Items[0]
	if !values.Private || !strings.Contains(values.Value, "password: hunter2") || !strings.Contains(values.Value, "replicas: 2") {
		t.Errorf("values %+v", values)
	}
	if !strings.Contains(secs[3].Items[0].Value, "# Source: web/templates/cm.yaml") {
		t.Errorf("manifest %q", secs[3].Items[0].Value)
	}
	if got := model.WithholdPrivate(secs)[2].Items[0].Value; got != model.Withheld {
		t.Errorf("values for agents %q", got)
	}
}

// The memo decodes a Secret once per resourceVersion, keeps values and manifests for the newest
// revision of each release only, decodes an older revision again when it becomes the newest, and
// forgets Secrets that are gone after two generations.
func TestDeriveMemo(t *testing.T) {
	m := newTestMemo()
	secrets := releaseSecrets(t)
	m.derive(secrets)
	v3 := types.UID("uid-apps-sh.helm.release.v1.web.v3")
	first := m.cur[v3]
	if first == nil || first.heavy == nil {
		t.Fatal("the newest revision is not kept with its values")
	}
	for uid, r := range m.cur {
		if uid != v3 && r.name == "web" && r.namespace == "apps" && r.heavy != nil {
			t.Errorf("%s keeps its values although it is not the newest", uid)
		}
	}
	m.derive(secrets)
	if m.cur[v3] != first {
		t.Error("an unchanged Secret was decoded again")
	}

	// A new resourceVersion is decoded again.
	changed := secrets[1].DeepCopy()
	changed.SetResourceVersion("2")
	m.derive(append([]*unstructured.Unstructured{changed}, secrets[0], secrets[2]))
	if m.cur[v3] == first || m.cur[v3].rv != "2" {
		t.Error("a changed Secret was not decoded again")
	}

	// The newest revision is deleted: revision 2 becomes the newest and gets its values back.
	objs := m.derive([]*unstructured.Unstructured{secrets[0], secrets[2]})
	if len(objs) != 1 {
		t.Fatalf("%d releases", len(objs))
	}
	if rev, _ := model.NestedInt64(objs[0], "revision"); rev != 2 {
		t.Errorf("revision %d after the newest was deleted", rev)
	}
	if r := m.cur["uid-apps-sh.helm.release.v1.web.v2"]; r == nil || r.heavy == nil {
		t.Error("the new newest revision has no values")
	}

	// Secrets that are gone leave the memo within two generations.
	for range 2 * memoGeneration {
		m.derive(secrets[:1])
	}
	if n := m.size(); n != 1 {
		t.Errorf("the memo holds %d revisions, want 1", n)
	}
}

// Hostile release Secrets never panic and never hide a release: one that does not decode shows
// with what its labels and name say, and the error.
func TestDeriveToleratesHostilePayloads(t *testing.T) {
	good := secretFor(t, testRelease{ns: "apps", name: "web", rev: 1, status: common.StatusDeployed})
	mk := func(name string, labels map[string]any, data any) *unstructured.Unstructured {
		o := good.DeepCopy()
		o.SetName(name)
		o.SetUID(types.UID("uid-" + name))
		o.Object["metadata"].(map[string]any)["labels"] = labels
		if data == nil {
			delete(o.Object, "data")
		} else {
			o.Object["data"] = data
		}
		return o
	}
	hostile := []*unstructured.Unstructured{
		mk("sh.helm.release.v1.api.v4", map[string]any{"owner": "helm", "name": "api", "version": "4", "status": "deployed"},
			map[string]any{"release": "!!!not base64"}),
		mk("sh.helm.release.v1.gz.v2", nil, map[string]any{"release": base64.StdEncoding.EncodeToString(
			[]byte(base64.StdEncoding.EncodeToString([]byte{0x1f, 0x8b, 0x08, 0, 1, 2, 3})))}),
		mk("sh.helm.release.v1.json.v1", map[string]any{"owner": "helm"}, map[string]any{"release": base64.StdEncoding.EncodeToString(
			[]byte(base64.StdEncoding.EncodeToString([]byte(`{"name": 7, "info": "x", "chart": [1]}`))))}),
		mk("weird", map[string]any{"owner": "helm", "version": "x"}, nil),
		mk("sh.helm.release.v1.data-not-map.v1", nil, "a string"),
		{Object: map[string]any{"metadata": "not a map"}},
		nil,
	}
	p := NewReleaseProfile()
	objs := p.Derive(hostile)
	got := byKey(objs)
	api := got["apps/api"]
	if api == nil || model.NestedString(api, "status") != "deployed" || !strings.Contains(model.NestedString(api, "error"), "base64") {
		t.Errorf("api from its labels: %v", api)
	}
	if got["apps/gz"] == nil || got["apps/weird"] == nil {
		t.Errorf("releases %v", got)
	}
	for _, o := range objs {
		model.BuildRow(p, "test", o)
		for _, s := range p.Detail(o) {
			_ = s
		}
	}
}

func FuzzDerive(f *testing.F) {
	f.Add([]byte(`{"name":"web","version":1,"info":{"status":"deployed"}}`), "web", "1", "deployed")
	f.Add([]byte("H4sI"), "", "x", "")
	f.Fuzz(func(_ *testing.T, payload []byte, name, version, status string) {
		o := &unstructured.Unstructured{Object: map[string]any{
			"metadata": map[string]any{"name": "sh.helm.release.v1." + name + ".v" + version, "namespace": "apps", "uid": "u",
				"labels": map[string]any{"owner": "helm", "name": name, "version": version, "status": status}},
			"data": map[string]any{"release": base64.StdEncoding.EncodeToString([]byte(base64.StdEncoding.EncodeToString(payload)))},
		}}
		p := NewReleaseProfile()
		for _, r := range p.Derive([]*unstructured.Unstructured{o, o}) {
			model.BuildRow(p, "fuzz", r)
			p.Detail(r)
		}
	})
}

func TestCutKeepsLinesAndSaysSo(t *testing.T) {
	text := strings.Repeat("line äöü\n", 1000)
	got := cut(text, 2000, 50)
	if strings.Count(got, "\n") > 51 || !strings.Contains(got, "of 1001 lines") {
		t.Errorf("cut: %d lines, ends %q", strings.Count(got, "\n"), got[len(got)-60:])
	}
	if cut("short", 100, 10) != "short" {
		t.Error("short text changed")
	}
}

func TestParseSecretName(t *testing.T) {
	for in, want := range map[string]string{
		"sh.helm.release.v1.web.v3":        "web 3",
		"sh.helm.release.v1.my.app.v12":    "my.app 12",
		"sh.helm.release.v1.web":           "web 0",
		"configmap":                        " 0",
		"sh.helm.release.v1.web.vbeta":     "web.vbeta 0",
		"sh.helm.release.v1.web-v2.v7":     "web-v2 7",
		"sh.helm.release.v1.with.v.dot.v2": "with.v.dot 2",
	} {
		n, r := parseSecretName(in)
		if got := fmt.Sprintf("%s %d", n, r); got != want {
			t.Errorf("%s: %q, want %q", in, got, want)
		}
	}
}

func TestStatusTokensAndPending(t *testing.T) {
	for status, want := range map[string]model.Status{
		"deployed": model.StatusOK, "failed": model.StatusError, "pending-install": model.StatusPending,
		"pending-upgrade": model.StatusPending, "pending-rollback": model.StatusPending, "uninstalling": model.StatusTerminating,
		"superseded": model.StatusNone, "uninstalled": model.StatusNone, "": model.StatusUnknown, "bogus": model.StatusUnknown,
	} {
		if got := StatusToken(status); got != want {
			t.Errorf("%q: %q, want %q", status, got, want)
		}
	}
	if !Pending("pending-rollback") || !Pending("uninstalling") || Pending("deployed") || Pending("failed") {
		t.Error("Pending")
	}
}
