package engine

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// secretReader is the API server's side of the drawer's GET: it serves the full Secrets of the
// source the cache watches (as metadata), with the same resourceVersions, and counts the GETs.
type secretReader struct {
	Actions
	src *testkit.MemorySource

	mu   sync.Mutex
	gets []string
}

func newSecretReader(src *testkit.MemorySource) *secretReader { return &secretReader{src: src} }

func (r *secretReader) GetSecret(_ context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	r.mu.Lock()
	r.gets = append(r.gets, namespace+"/"+name)
	r.mu.Unlock()
	// The source lists only through the plain List; the engine may not import client-go's
	// tools/cache for the context adapter.
	list, err := r.src.ListerWatcher(cache.Scope{GVR: testkit.SecretsGVR}).List(metav1.ListOptions{}) //nolint:staticcheck // see above
	if err != nil {
		return nil, err
	}
	for _, it := range list.(*unstructured.UnstructuredList).Items {
		if it.GetNamespace() == namespace && it.GetName() == name {
			return it.DeepCopy(), nil
		}
	}
	return nil, errors.New(`secrets "` + name + `" not found`)
}

func (r *secretReader) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.gets)
}

// secretsSession connects a session whose Secrets are the real, metadata-only profile, with r as
// the cluster's reader.
func secretsSession(t *testing.T, src *testkit.MemorySource, r *secretReader) *Session {
	t.Helper()
	p, _ := kinds.ByKey(secretsKind)
	s, fc := newTestSession(t, src, 5*time.Millisecond)
	fc.profiles = append([]*model.KindProfile{p}, testPods, testEvents, testNamespaces)
	fc.actions = r
	connectTest(t, s)
	return s
}

// waitData waits until the drawer model of ref has its Data section filled.
func waitData(t *testing.T, s *Session, ref model.ObjectRef) *DetailModel {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var m *DetailModel
	if !testkit.WaitFor(ctx, func() bool {
		var err error
		m, err = s.Detail(ref)
		if err != nil || m.Loading {
			return false
		}
		v := sectionValue(m, "Data")
		return v != "" && v != "Loading…"
	}) {
		t.Fatalf("no data for %s: %+v", ref, m)
	}
	return m
}

func dataItems(m *DetailModel) []model.DetailItem {
	for _, sec := range m.Sections {
		if sec.Title == "Data" {
			return sec.Items
		}
	}
	return nil
}

// The Secrets drawer: one GET per open of a Secret, values masked in the model (for the GUI and
// agents alike), a binary value by its size only, the Type from the GET, and the data dropped when
// the drawer moves or closes; the list itself never fetches data.
func TestSecretDrawerFetchesOncePerOpenAndMasks(t *testing.T) {
	src := testkit.NewMemorySource()
	r := newSecretReader(src)
	db := testkit.Secret("apps", "db", map[string]string{"app": "db"}, map[string]string{"password": "hunter2", "url": "postgres://db:5432"})
	_ = unstructured.SetNestedField(db.Object, base64Of("\xff\xfe\x00\x01"), "data", "blob")
	src.Set(testkit.SecretsGVR, db)
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "tls", nil, map[string]string{"tls.key": "-----BEGIN KEY-----"}))
	s := secretsSession(t, src, r)

	s.UpdateUI(func(u *UIState) { u.View = secretsKind })
	snap := waitTable(t, s, secretsKind, func(ts *TableSnapshot) bool { return ts.Synced && len(ts.Rows) == 2 })
	if r.count() != 0 {
		t.Fatalf("the list fetched Secret data: %d GETs", r.count())
	}
	if b, _ := json.Marshal(snap); strings.Contains(string(b), "hunter2") || strings.Contains(string(b), base64Of("hunter2")) {
		t.Fatal("a value reached the table")
	}

	db0 := model.ObjectRef{Context: "test", Kind: secretsKind, Namespace: "apps", Name: "db"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &db0 })
	m := waitData(t, s, db0)
	if r.count() != 1 {
		t.Errorf("%d GETs for one open, want 1", r.count())
	}
	items := dataItems(m)
	if len(items) != 3 || items[0].Name != "blob" || items[1].Name != "password" || items[2].Name != "url" {
		t.Fatalf("data items %+v", items)
	}
	if items[0].Value != "binary data, 4 bytes" || len(items[0].Actions) != 0 {
		t.Errorf("binary item %+v", items[0])
	}
	if items[1].Value != MaskedValue+" (7 bytes)" || len(items[1].Actions) != 1 || items[1].Actions[0].Command != RevealCommand {
		t.Errorf("masked item %+v", items[1])
	}
	var rp RevealParams
	if err := json.Unmarshal(items[1].Actions[0].Params, &rp); err != nil || rp.Key != "password" || !rp.Ref.Same(db0) {
		t.Errorf("reveal params %s", items[1].Actions[0].Params)
	}
	if got := propertyValue(m, "Type"); got != "Opaque" {
		t.Errorf("Type %q", got)
	}
	for _, origin := range []Origin{OriginGUI, OriginExternalAgent} {
		mm, err := s.DetailWait(context.WithValue(context.Background(), originKey{}, origin), db0)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(mm)
		if strings.Contains(string(b), "hunter2") || strings.Contains(string(b), base64Of("hunter2")) || strings.Contains(string(b), "5432") {
			t.Errorf("%s: a value reached the drawer model", origin)
		}
	}

	v, err := s.SecretValue(db0, "password")
	if err != nil || v.Value != "hunter2" || v.Size != 7 || v.Cut {
		t.Errorf("SecretValue %+v, %v", v, err)
	}
	if _, err := s.SecretValue(db0, "blob"); err == nil {
		t.Error("a binary value was revealed")
	}

	// A change of the Secret's metadata refreshes the drawer, not the GET; the drawer says the data
	// may be stale.
	db.SetLabels(map[string]string{"app": "db", "rotated": "yes"})
	src.Set(testkit.SecretsGVR, db)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool {
		m, _ := s.Detail(db0)
		return strings.Contains(sectionText(m, "Data"), "changed since its drawer opened")
	}) {
		t.Error("no note that the Secret changed")
	}
	if r.count() != 1 {
		t.Errorf("%d GETs after a watch event, want 1", r.count())
	}

	// Moving the drawer drops the data and fetches the other Secret; closing drops that too.
	tls := model.ObjectRef{Context: "test", Kind: secretsKind, Namespace: "apps", Name: "tls"}
	s.UpdateUI(func(u *UIState) { u.Drawer = &tls })
	waitData(t, s, tls)
	if _, err := s.SecretValue(db0, "password"); !errors.Is(err, ErrNoSecretData) {
		t.Errorf("the first Secret's data survived the move: %v", err)
	}
	s.UpdateUI(func(u *UIState) { u.Drawer = nil })
	if _, err := s.SecretValue(tls, "tls.key"); !errors.Is(err, ErrNoSecretData) {
		t.Errorf("the data survived closing the drawer: %v", err)
	}
	if r.count() != 2 {
		t.Errorf("%d GETs, want 2", r.count())
	}
	// A Secret the drawer does not show: object_get says where its data is, and fetches nothing.
	mm, err := s.DetailWait(context.Background(), db0)
	if err != nil || !strings.Contains(sectionValue(mm, "Data"), "only while its drawer is open") || r.count() != 2 {
		t.Errorf("object_get of a closed Secret: %q, %d GETs", sectionValue(mm, "Data"), r.count())
	}
	// Opening it again is a new open: one more GET.
	s.UpdateUI(func(u *UIState) { u.Drawer = &db0 })
	waitData(t, s, db0)
	if r.count() != 3 {
		t.Errorf("%d GETs after opening again, want 3", r.count())
	}
}

// A GET that fails, a cluster connection that cannot read Secrets and a known denial show as the
// Data section's state, never as a crash or a value.
func TestSecretDrawerStates(t *testing.T) {
	src := testkit.NewMemorySource()
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "ghost", nil, map[string]string{"k": "v"}))
	r := newSecretReader(testkit.NewMemorySource()) // the server no longer has it
	s := secretsSession(t, src, r)
	ghost := model.ObjectRef{Context: "test", Kind: secretsKind, Namespace: "apps", Name: "ghost"}
	s.UpdateUI(func(u *UIState) { u.View, u.Drawer = secretsKind, &ghost })
	m := waitData(t, s, ghost)
	if got := sectionValue(m, "Data"); !strings.Contains(got, "could not be read") || !strings.Contains(got, "not found") {
		t.Errorf("failed GET: %q", got)
	}

	b := s.Backend()
	b.Access = denyAll{}
	s.UpdateUI(func(u *UIState) { u.Drawer = nil })
	s.UpdateUI(func(u *UIState) { u.Drawer = &ghost })
	m = waitData(t, s, ghost)
	if got := sectionValue(m, "Data"); !strings.Contains(got, "not permitted: you may not get secrets in apps") {
		t.Errorf("denied GET: %q", got)
	}
	if r.count() != 1 {
		t.Errorf("%d GETs, want 1 (the denied open asks nothing)", r.count())
	}
}

type denyAll struct{}

func (denyAll) Allowed(string, Verb) (bool, bool) { return false, true }

// A long value is kept cut to the reveal caps, a multi-byte text cut at a character boundary.
func TestSecretValueCaps(t *testing.T) {
	long := strings.Repeat("ä", 3000) // 6000 bytes
	k := newSecretKey("big", base64Of(long))
	if k.binary || len(k.value) > secretValueBytes || k.size != 6000 || !strings.HasPrefix(long, string(k.value)) {
		t.Errorf("long value: binary %v, kept %d of %d", k.binary, len(k.value), k.size)
	}
	lines := strings.Repeat("line\n", 100)
	if k := newSecretKey("lines", base64Of(lines)); strings.Count(string(k.value), "\n") >= secretValueLines {
		t.Errorf("kept %d lines", strings.Count(string(k.value), "\n"))
	}
	if k := newSecretKey("bell", base64Of("ring\a")); !k.binary || k.value != nil {
		t.Errorf("control characters: %+v", k)
	}
}

func base64Of(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// propertyValue returns the value of a Properties item.
func propertyValue(m *DetailModel, name string) string {
	if len(m.Sections) == 0 {
		return ""
	}
	for _, it := range m.Sections[0].Items {
		if it.Name == name {
			return it.Value
		}
	}
	return ""
}

// sectionText joins every value of a section.
func sectionText(m *DetailModel, title string) string {
	if m == nil {
		return ""
	}
	var out []string
	for _, sec := range m.Sections {
		if sec.Title == title {
			for _, it := range sec.Items {
				out = append(out, it.Value)
			}
		}
	}
	return strings.Join(out, "\n")
}
