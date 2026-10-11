package fyneui

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// The Secrets view and drawer as the engine serves them: rows from the metadata the cache holds
// (Keys and Type empty), a drawer whose Data lists the keys masked with Show beside each, and
// secret.reveal answering the GUI with one value. Every name and value is made up.

// fixtureSecretObjects are Secrets as the metadata-only cache holds them.
func fixtureSecretObjects() []*unstructured.Unstructured {
	var out []*unstructured.Unstructured
	for i, s := range []struct {
		ns, name string
		labels   map[string]any
	}{
		{"checkout", "checkout-db", map[string]any{"app.kubernetes.io/name": "checkout-api", "app.kubernetes.io/component": "database"}},
		{"checkout", "checkout-api-token", map[string]any{"app.kubernetes.io/name": "checkout-api"}},
		{"payments", "stripe-webhook", map[string]any{"team": "payments"}},
		{"storefront", "storefront-tls", map[string]any{"cert-manager.io/certificate-name": "storefront"}},
		{"storefront", "sh.helm.release.v1.storefront.v7", map[string]any{"owner": "helm", "name": "storefront", "status": "deployed", "version": "7"}},
		{"kube-system", "bootstrap-token-k8m2xq", nil},
		{"monitoring", "grafana-admin", map[string]any{"app.kubernetes.io/name": "grafana"}},
	} {
		meta := map[string]any{"name": s.name, "namespace": s.ns, "uid": "uid-sec-" + s.name, "resourceVersion": "4821" + string(rune('0'+i)),
			"creationTimestamp": ts(time.Duration(3+i*5) * 24 * time.Hour)}
		if s.labels != nil {
			meta["labels"] = s.labels
		}
		out = append(out, &unstructured.Unstructured{Object: map[string]any{
			"apiVersion": "meta.k8s.io/v1", "kind": "PartialObjectMetadata", "metadata": meta,
		}})
	}
	return out
}

// secretsFake is a fake connected to a cluster with Secrets, the drawer open on checkout-db with
// its data fetched, and secret.reveal answering with the value of a key.
func secretsFake(t testing.TB) (*fakeEngine, model.ObjectRef) {
	f := kindsFake(t, "secrets")
	f.kinds = append(f.kinds, kindInfo{Kind: "secrets", Title: "Secrets", APIKind: "Secret", Version: "v1", Namespaced: true})
	objs := fixtureSecretObjects()
	f.tables["secrets"] = kindSnapshot(t, "secrets", objs)
	p, _ := kinds.ByKey("secrets")
	ref := model.ObjectRef{Context: "demo", Kind: "secrets", Namespace: "checkout", Name: "checkout-db"}
	m, err := engine.BuildDetail(p, ref, objs[0], nil, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	// What the engine adds from the drawer's GET (secretdata.go): the Type, and the keys masked.
	m.Sections[0].Items = append(m.Sections[0].Items, model.DetailItem{Name: "Type", Value: "Opaque"})
	values := map[string]string{
		"DB_HOST": "checkout-db.checkout.svc.cluster.local", "DB_PASSWORD": "correct-horse-battery-staple",
		"DB_USER": "checkout", "ca.crt": "", "DATABASE_URL": "postgres://checkout:correct-horse-battery-staple@checkout-db:5432/orders?sslmode=verify-full&application_name=checkout-api",
	}
	data := model.DetailSection{Title: "Data"}
	for _, k := range []string{"DATABASE_URL", "DB_HOST", "DB_PASSWORD", "DB_USER", "ca.crt"} {
		if k == "ca.crt" {
			data.Items = append(data.Items, model.DetailItem{Name: k, Value: "binary data, 1.2 KiB"})
			continue
		}
		params, _ := json.Marshal(engine.RevealParams{Ref: ref, Key: k})
		data.Items = append(data.Items, model.DetailItem{
			Name: k, Value: engine.MaskedValue + " (" + byteText(len(values[k])) + ")",
			Actions: []model.ItemAction{{Label: "Show", Command: engine.RevealCommand, Params: params}},
		})
	}
	m.Sections = append(m.Sections, data)
	f.details[ref.String()] = m
	f.handlers[engine.RevealCommand] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var p engine.RevealParams
		if err := json.Unmarshal(inv.Params, &p); err != nil {
			return nil, err
		}
		if inv.Origin != engine.OriginGUI {
			t.Errorf("secret.reveal dispatched by %s", inv.Origin)
		}
		return engine.SecretValue{Key: p.Key, Value: values[p.Key], Size: len(values[p.Key])}, nil
	}
	f.ui.View = "secrets"
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	return f, ref
}

// dataButtons returns the drawer's buttons in its Data section, by label.
func drawerButtons(d *drawer) map[string][]*widget.Button {
	out := map[string][]*widget.Button{}
	for _, b := range findAll[*widget.Button](d.details) {
		out[b.Text] = append(out[b.Text], b)
	}
	return out
}

// drawerHasText reports whether a label or text line of the drawer's details reads s.
func drawerHasText(d *drawer, s string) bool {
	for _, l := range findAll[*widget.Label](d.details) {
		if strings.Contains(l.Text, s) {
			return true
		}
	}
	for _, l := range findAll[*textLine](d.details) {
		if strings.Contains(l.text, s) {
			return true
		}
	}
	return false
}

// Show reveals one value in place, from the GUI; Hide masks it again; moving the drawer forgets
// it. The value never reaches a toast.
func TestSecretRevealInDrawer(t *testing.T) {
	f, ref := secretsFake(t)
	sh, _, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && len(drawerButtons(sh.drawer)["Show"]) == 4 })
	if drawerHasText(sh.drawer, "correct-horse") {
		t.Fatal("a value shows before Show")
	}
	test.Tap(drawerButtons(sh.drawer)["Show"][2]) // DB_PASSWORD
	q.until(t, "the value", func() bool { return drawerHasText(sh.drawer, "correct-horse-battery-staple") })
	if calls := f.callsOf(engine.RevealCommand); len(calls) != 1 || calls[0].Origin != engine.OriginGUI ||
		!strings.Contains(string(calls[0].Params), `"key":"DB_PASSWORD"`) {
		t.Errorf("reveal calls %+v", calls)
	}
	if b := drawerButtons(sh.drawer); len(b["Hide"]) != 1 || len(b["Show"]) != 3 {
		t.Errorf("buttons after Show: %v", b)
	}
	for _, o := range sh.toasts.Objects {
		for _, l := range findAll[*textLine](o) {
			if strings.Contains(l.text, "correct-horse") {
				t.Error("the value reached a toast")
			}
		}
	}

	test.Tap(drawerButtons(sh.drawer)["Hide"][0])
	q.until(t, "the value to hide", func() bool { return !drawerHasText(sh.drawer, "correct-horse") })

	test.Tap(drawerButtons(sh.drawer)["Show"][2])
	q.until(t, "the value again", func() bool { return drawerHasText(sh.drawer, "correct-horse") })
	other := model.ObjectRef{Context: "demo", Kind: "secrets", Namespace: "payments", Name: "stripe-webhook"}
	f.update(func(u *engine.UIState) { u.Selected, u.Drawer = []model.ObjectRef{other}, &other })
	q.until(t, "the other Secret", func() bool { return sh.drawer.shown != nil && *sh.drawer.shown == other })
	if len(sh.drawer.revealed) != 0 {
		t.Error("a revealed value survived the move to another object")
	}
	f.update(func(u *engine.UIState) { u.Selected, u.Drawer = []model.ObjectRef{ref}, &ref })
	q.until(t, "the first Secret", func() bool { return sh.drawer.last != nil && sh.drawer.last.Ref == ref })
	if drawerHasText(sh.drawer, "correct-horse") {
		t.Error("the value shows again without Show")
	}
}

// renderSecrets renders the Secrets list with the drawer open, one value revealed.
func renderSecrets(t *testing.T, mode ThemeMode, name string) {
	f, _ := secretsFake(t)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the drawer", func() bool {
		return len(sh.kinds) > 0 && sh.view.snap != nil && sh.drawer.last != nil && len(drawerButtons(sh.drawer)["Show"]) == 4 && sh.diag.total > 0
	})
	test.Tap(drawerButtons(sh.drawer)["Show"][0]) // DATABASE_URL: long, wraps
	q.until(t, "the value", func() bool { return drawerHasText(sh.drawer, "sslmode") })
	settle(w)
	savePNG(t, w, name)
}

func TestRenderSecretsDark(t *testing.T)  { renderSecrets(t, ThemeDark, "secrets_dark.png") }
func TestRenderSecretsLight(t *testing.T) { renderSecrets(t, ThemeLight, "secrets_light.png") }

var _ fyne.CanvasObject = (*widget.Label)(nil)
