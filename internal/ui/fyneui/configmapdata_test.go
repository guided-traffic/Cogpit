package fyneui

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// A Config Map's Data in the drawer (the owner's F3): each key above its value's editor, values
// edited and saved through configMap.reviewData and configMap.saveData, a conflict shown before
// anything is written over a newer version, edited values kept when the drawer moves on. The keys
// are Cilium's option names (the owner's capture); the values are made up.

var ciliumRef = model.ObjectRef{Context: "demo", Kind: "configmaps", Namespace: "kube-system", Name: "cilium-config"}

// ciliumValues are the fixture Config Map's data.
var ciliumValues = map[string]string{
	"agent-not-ready-taint-key":                   "node.cilium.io/agent-not-ready",
	"allow-localhost":                             "auto",
	"auto-direct-node-routes":                     "false",
	"bpf-distributed-lru":                         "false",
	"bpf-events-drop-enabled":                     "true",
	"bpf-lb-acceleration":                         "disabled",
	"bpf-lb-map-max":                              "65536",
	"bpf-lb-source-range-all-types":               "false",
	"bpf-map-dynamic-size-ratio":                  "0.0025",
	"clustermesh-enable-endpoint-sync":            "false",
	"controller-group-metrics":                    "write-cni-file sync-host-ips sync-lb-maps-with-k8s-services",
	"hubble-flowlogs-config.yaml":                 "flowLogs:\n  - name: all\n    fieldMask: []\n    filePath: /var/run/cilium/hubble/events.log\n",
	"kube-proxy-replacement-healthz-bind-address": "",
}

// cmFake answers the Config Map data commands as the engine does: the values from its state,
// a review with the diff and an accepted dry run, a save that stores the values under a new
// version; stale and conflict make the next review or save meet a newer version.
type cmFake struct {
	mu       sync.Mutex
	values   map[string]string
	rv       int
	stale    string // the review answers stale at this version
	conflict string // the save answers with a conflict at this version, unless based on it
	readOnly string
}

func newCMFake(t testing.TB, f *fakeEngine) *cmFake { return newCMFakeWith(t, f, ciliumValues) }

// newCMFakeWith is newCMFake with the Config Map holding values.
func newCMFakeWith(t testing.TB, f *fakeEngine, values map[string]string) *cmFake {
	c := &cmFake{values: map[string]string{}, rv: 4211}
	for k, v := range values {
		c.values[k] = v
	}
	f.kinds = append(f.kinds, kindInfo{Kind: "configmaps", Title: "Config Maps", APIKind: "ConfigMap", Version: "v1", Namespaced: true})
	obj := c.object()
	f.tables["configmaps"] = kindSnapshot(t, "configmaps", []*unstructured.Unstructured{obj})
	p, _ := kinds.ByKey("configmaps")
	m, err := engine.BuildDetail(p, ciliumRef, obj, nil, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	f.details[ciliumRef.String()] = m
	params := func(inv engine.Invocation) (configMapDataParams, error) {
		var p configMapDataParams
		err := json.Unmarshal(normalize(inv.Params), &p)
		return p, err
	}
	f.handlers["configMap.readData"] = func(context.Context, engine.Invocation) (any, error) {
		c.mu.Lock()
		defer c.mu.Unlock()
		d := engine.ConfigMapData{Ref: ciliumRef, ResourceVersion: strconv.Itoa(c.rv), ReadOnly: c.readOnly}
		for _, k := range c.keys() {
			d.Values = append(d.Values, engine.ConfigMapValue{Key: k, Value: c.values[k]})
		}
		return d, nil
	}
	f.handlers["configMap.reviewData"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		p, err := params(inv)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		lines := engine.DiffLines(c.text(nil), c.text(p.Data))
		r := engine.ConfigMapDataReview{Ref: ciliumRef, Changed: engine.DiffChanged(lines), Lines: engine.CompactDiff(lines, 3),
			BaseResourceVersion: p.ResourceVersion, CachedResourceVersion: strconv.Itoa(c.rv)}
		for k, v := range p.Data {
			if c.values[k] != v {
				r.Keys = append(r.Keys, k)
			}
		}
		for _, l := range lines {
			switch l.Op {
			case engine.DiffAdded:
				r.Added++
			case engine.DiffRemoved:
				r.Removed++
			}
		}
		switch {
		case c.stale != "":
			r.Stale, r.CachedResourceVersion = true, c.stale
		case r.Changed:
			r.Accepted = true
		}
		return r, nil
	}
	f.handlers["configMap.saveData"] = func(_ context.Context, inv engine.Invocation) (any, error) {
		p, err := params(inv)
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.conflict != "" && p.ResourceVersion != c.conflict {
			lines := engine.DiffLines(c.text(nil), c.text(p.Data))
			return nil, &engine.DataConflictError{Ref: ciliumRef, BaseResourceVersion: p.ResourceVersion,
				CurrentResourceVersion: c.conflict, Diff: engine.CompactDiff(lines, 3)}
		}
		var keys []string
		for k, v := range p.Data {
			c.values[k] = v
			keys = append(keys, k)
		}
		c.rv++
		sort.Strings(keys)
		return engine.ConfigMapDataSaved{Ref: ciliumRef, Keys: keys, ResourceVersion: strconv.Itoa(c.rv)}, nil
	}
	// The save command in the fake's registry, so that the drawer's check finds it enabled.
	if err := engine.Register(f.reg, engine.Spec[configMapDataParams]{ID: "configMap.saveData", Label: "Save Config Map Data",
		Danger: engine.DangerMutating, Verbs: nil,
		Enabled: func(*engine.Session, configMapDataParams) error { return f.reason("configMap.saveData") },
		Targets: func(_ *engine.Session, p configMapDataParams) ([]model.ObjectRef, error) {
			return []model.ObjectRef{p.Ref}, nil
		},
		Run: func(context.Context, *engine.Session, configMapDataParams) (any, error) { return nil, nil }}); err != nil {
		t.Fatal(err)
	}
	f.ui.View = "configmaps"
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ciliumRef}, &ciliumRef
	return c
}

func (c *cmFake) keys() []string {
	keys := make([]string, 0, len(c.values))
	for k := range c.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// object is the Config Map as the cache would hold it.
func (c *cmFake) object() *unstructured.Unstructured {
	data := map[string]any{}
	for k, v := range c.values {
		data[k] = v
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "ConfigMap",
		"metadata": map[string]any{"name": ciliumRef.Name, "namespace": ciliumRef.Namespace, "uid": "uid-cilium-config",
			"resourceVersion": strconv.Itoa(c.rv), "creationTimestamp": ts(169 * 24 * time.Hour),
			"labels": map[string]any{"app.kubernetes.io/managed-by": "Helm", "app.kubernetes.io/part-of": "cilium", "io.cilium/app": "config"}},
		"data": data,
	}}
}

// text is the data as YAML-like lines with over applied, for the fake's diffs.
func (c *cmFake) text(over map[string]string) string {
	var b strings.Builder
	b.WriteString("data:\n")
	for _, k := range c.keys() {
		v := c.values[k]
		if o, ok := over[k]; ok {
			v = o
		}
		b.WriteString("  " + k + ": " + strconv.Quote(v) + "\n")
	}
	return b.String()
}

// cmShell opens the drawer on cilium-config and waits for its values.
func cmShell(t *testing.T, mode ThemeMode) (*shell, fyne.Window, *mainQueue, *fakeEngine, *cmFake) {
	t.Helper()
	f := kindsFake(t, "configmaps")
	c := newCMFake(t, f)
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the Config Map's values", func() bool {
		cd := cmDataOf(sh)
		return cd != nil && cd.loaded && len(sh.kinds) > 0 && sh.view.snap != nil
	})
	settle(w)
	return sh, w, q, f, c
}

func cmDataOf(sh *shell) *cmData {
	cd, _ := sh.drawer.kept[cmDataKept].(*cmData)
	return cd
}

// TestConfigMapDrawerKeysAboveValues (F3): each key on a line of its own, its value below it in an
// editor the width of the details, with line numbers and the value's colours; editable once read.
func TestConfigMapDrawerKeysAboveValues(t *testing.T) {
	sh, _, _, _, _ := cmShell(t, ThemeDark)
	cd := cmDataOf(sh)
	if len(cd.keys) != len(ciliumValues) || cd.keys[0] != "agent-not-ready-taint-key" {
		t.Fatalf("keys %v", cd.keys)
	}
	width := cd.list.Size().Width
	if width < 300 {
		t.Fatalf("the list is %v wide", width)
	}
	for _, k := range cd.keys {
		r := cd.rows[k]
		name, value := r.name.Position(), r.value.Position()
		if name.X != 0 || value.X != 0 || value.Y < name.Y+r.name.Size().Height || r.value.Size().Width != width {
			t.Errorf("%s: key at %v %v, value at %v %v; want the key above the value at the full width %v",
				k, name, r.name.Size(), value, r.value.Size(), width)
		}
		if r.view.ReadOnly() || !r.view.LineNumbers() || r.view.Text() != ciliumValues[k] {
			t.Errorf("%s: read-only %v, line numbers %v, text %q", k, r.view.ReadOnly(), r.view.LineNumbers(), r.view.Text())
		}
	}
	// A one-line value's editor is three lines high, a five-line value's five.
	one, five := cd.rows["allow-localhost"], cd.rows["hubble-flowlogs-config.yaml"]
	if one.frame.Size().Height != one.view.HeightFor(cmMinLines, width) || five.frame.Size().Height != five.view.HeightFor(5, width) {
		t.Errorf("editor heights %v and %v", one.frame.Size().Height, five.frame.Size().Height)
	}
	if !cd.buttons.Visible() || !cd.save.Disabled() || !cd.revert.Disabled() || cd.status.Visible() {
		t.Error("the footer of unedited values: Save and Revert disabled, no status")
	}
}

// TestConfigMapDrawerGripResizes: the grip at an editor's bottom right drags it taller, up to
// cmMaxDrag lines, and shorter, down to one.
func TestConfigMapDrawerGripResizes(t *testing.T) {
	sh, w, _, _, _ := cmShell(t, ThemeDark)
	cd := cmDataOf(sh)
	r := cd.rows["allow-localhost"]
	grip := findAll[*cmGrip](r.frame)[0]
	before := r.frame.MinSize().Height
	grip.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, 60)})
	settle(w)
	if got := r.frame.Size().Height; got != before+60 {
		t.Errorf("dragged 60 down: %v, want %v", got, before+60)
	}
	grip.Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(0, -1000)})
	if got := r.frame.MinSize().Height; got != r.view.HeightFor(1, cd.width) {
		t.Errorf("dragged far up: %v, want one line", got)
	}
}

// TestConfigMapDrawerEditAndSave: typing in two editors enables Save and Revert; Save reviews the
// two values (diff, dry run) in a dialog based on the version read, and its Save writes exactly
// them through configMap.saveData.
func TestConfigMapDrawerEditAndSave(t *testing.T) {
	sh, w, q, f, c := cmShell(t, ThemeDark)
	cd := cmDataOf(sh)
	typeInto(cd.rows["allow-localhost"], "always")
	typeInto(cd.rows["bpf-lb-map-max"], "131072")
	if cd.save.Disabled() || cd.revert.Disabled() || !strings.Contains(cd.status.text, "2 values, based on version 4211") {
		t.Fatalf("after editing: save %v revert %v status %q", cd.save.Disabled(), cd.revert.Disabled(), cd.status.text)
	}
	test.Tap(cd.save)
	q.until(t, "the review", func() bool { return cd.review != nil && !cd.review.save.Disabled() })
	if calls := f.callsOf("configMap.reviewData"); len(calls) != 1 || !strings.Contains(string(calls[0].Params), `"resourceVersion":"4211"`) {
		t.Fatalf("review calls %+v", calls)
	}
	if !strings.Contains(cd.review.banner.text, "2 values") || !strings.Contains(cd.review.status.text, "accepts") || len(cd.review.diff.rows) == 0 {
		t.Errorf("the review: %q / %q, %d diff rows", cd.review.banner.text, cd.review.status.text, len(cd.review.diff.rows))
	}
	settle(w)
	test.Tap(findButton(w, "Save"))
	q.until(t, "the save", func() bool { return cd.review == nil && strings.HasPrefix(cd.status.text, "Saved") })
	calls := f.callsOf("configMap.saveData")
	if len(calls) != 1 {
		t.Fatalf("save calls %+v", calls)
	}
	p := decodeParams[configMapDataParams](t, calls[0])
	if p.ResourceVersion != "4211" || len(p.Data) != 2 || p.Data["allow-localhost"] != "always" || p.Data["bpf-lb-map-max"] != "131072" || calls[0].Origin != engine.OriginGUI {
		t.Errorf("save params %+v from %s", p, calls[0].Origin)
	}
	if cd.dirty() || !cd.save.Disabled() || cd.status.text != "Saved 2 values · version 4212" {
		t.Errorf("after the save: dirty %v, save disabled %v, status %q", cd.dirty(), cd.save.Disabled(), cd.status.text)
	}
	if c.values["allow-localhost"] != "always" {
		t.Error("the fake did not store the value")
	}
}

// TestConfigMapDrawerConflict: the Config Map changed on the cluster since the values were read.
// The review shows the values against the newer version and writes nothing; Save over version N
// is the explicit second step, based on N. A save that meets a still newer version shows that.
func TestConfigMapDrawerConflict(t *testing.T) {
	sh, w, q, f, c := cmShell(t, ThemeDark)
	cd := cmDataOf(sh)
	typeInto(cd.rows["allow-localhost"], "always")
	c.mu.Lock()
	c.stale, c.conflict = "4300", "4300"
	c.mu.Unlock()
	test.Tap(cd.save)
	q.until(t, "the stale review", func() bool { return cd.review != nil && cd.review.over == "4300" })
	if len(f.callsOf("configMap.saveData")) != 0 || !strings.Contains(cd.review.banner.text, "changed on the cluster since version 4211") ||
		cd.review.save.Text != "Save over version 4300" {
		t.Fatalf("the stale review: %q, button %q", cd.review.banner.text, cd.review.save.Text)
	}

	// Meanwhile the Config Map moved again: the save based on 4300 meets 4400.
	c.mu.Lock()
	c.conflict = "4400"
	c.mu.Unlock()
	settle(w)
	test.Tap(cd.review.save)
	q.until(t, "the conflict", func() bool { return cd.review != nil && cd.review.over == "4400" })
	if calls := f.callsOf("configMap.saveData"); len(calls) != 1 || decodeParams[configMapDataParams](t, calls[0]).ResourceVersion != "4300" {
		t.Fatalf("the save over 4300: %+v", calls)
	}
	test.Tap(cd.review.save)
	q.until(t, "the save over 4400", func() bool { return cd.review == nil && strings.HasPrefix(cd.status.text, "Saved") })
	if calls := f.callsOf("configMap.saveData"); len(calls) != 2 || decodeParams[configMapDataParams](t, calls[1]).ResourceVersion != "4400" {
		t.Errorf("the second save: %+v", calls)
	}
}

// TestConfigMapDrawerKeepsUnsavedEdits: the drawer moving to another object keeps the edited
// values and says so; back on the Config Map they are in the editors again, based on the version
// they were edited at. Revert drops them.
func TestConfigMapDrawerKeepsUnsavedEdits(t *testing.T) {
	sh, _, q, f, _ := cmShell(t, ThemeDark)
	typeInto(cmDataOf(sh).rows["allow-localhost"], "always")
	other := f.tables["pods"].Rows[0].Ref
	f.update(func(u *engine.UIState) { u.Selected, u.Drawer = []model.ObjectRef{other}, &other })
	q.until(t, "the other object", func() bool { return sh.drawer.last != nil && sh.drawer.last.Ref == other })
	if u := sh.drawer.unsaved[ciliumRef.String()]; u == nil || u.base != "4211" || u.values["allow-localhost"] != "always" {
		t.Fatalf("kept %+v", u)
	}
	if !toastSays(sh, "unsaved changes to ConfigMap kube-system/cilium-config are kept") {
		t.Error("no toast says the changes are kept")
	}

	f.update(func(u *engine.UIState) { u.Selected, u.Drawer = []model.ObjectRef{ciliumRef}, &ciliumRef })
	q.until(t, "the Config Map again", func() bool { cd := cmDataOf(sh); return cd != nil && cd.loaded && cd.dirty() })
	cd := cmDataOf(sh)
	if cd.rows["allow-localhost"].view.Text() != "always" || cd.base != "4211" || !strings.Contains(cd.status.text, "are back") {
		t.Errorf("back: %q based on %s, status %q", cd.rows["allow-localhost"].view.Text(), cd.base, cd.status.text)
	}
	if len(sh.drawer.unsaved) != 0 {
		t.Error("the kept values stay kept after they came back")
	}
	test.Tap(cd.revert)
	if cd.dirty() || cd.rows["allow-localhost"].view.Text() != "auto" || !cd.save.Disabled() {
		t.Error("Revert did not put the value back")
	}
}

// TestConfigMapDrawerReadOnly: an immutable Config Map, and a save the cluster does not allow
// (the demo cluster, the access reviews), leave the editors read-only and say why.
func TestConfigMapDrawerReadOnly(t *testing.T) {
	f := kindsFake(t, "configmaps")
	c := newCMFake(t, f)
	c.readOnly = "The Config Map is immutable: its data cannot change."
	sh, _, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the values", func() bool { cd := cmDataOf(sh); return cd != nil && cd.loaded })
	cd := cmDataOf(sh)
	if !cd.rows["allow-localhost"].view.ReadOnly() || !cd.save.Disabled() || cd.status.text != "Read-only: "+c.readOnly {
		t.Errorf("immutable: read-only %v, save disabled %v, status %q", cd.rows["allow-localhost"].view.ReadOnly(), cd.save.Disabled(), cd.status.text)
	}

	f2 := kindsFake(t, "configmaps")
	newCMFake(t, f2)
	f2.disabled["configMap.saveData"] = "not permitted: update configmaps in namespace kube-system"
	sh2, _, q2 := newTestShell(t, f2, Options{Theme: ThemeDark})
	q2.until(t, "the values", func() bool { cd := cmDataOf(sh2); return cd != nil && cd.loaded })
	cd2 := cmDataOf(sh2)
	if !cd2.rows["allow-localhost"].view.ReadOnly() || !strings.Contains(cd2.status.text, "not permitted") {
		t.Errorf("not permitted: read-only %v, status %q", cd2.rows["allow-localhost"].view.ReadOnly(), cd2.status.text)
	}
}

// TestDrawerNameWrapsAtSeparators (F3): a name without spaces wider than the name column wraps
// after - . _ / within it, and by characters where it has none; a name that fits stays one line.
func TestDrawerNameWrapsAtSeparators(t *testing.T) {
	test.NewTempApp(t)
	size := theme.DefaultTheme().Size(theme.SizeNameText)
	width := float32(128)
	for _, c := range []struct {
		name string
		want []string
	}{
		{"Created", []string{"Created"}},
		{"Pod Management Policy", nil}, // at its spaces, as before
		{"agent-not-ready-taint-key", nil},
		{"node.cilium.io/agent-not-ready", nil},
		{"MAX_DATABASE_CONNECTIONS_PER_POOL", nil},
		{strings.Repeat("x", 60), nil},
	} {
		lines := wrapName(c.name, width, size, fyne.TextStyle{})
		if c.want != nil && !slices.Equal(lines, c.want) {
			t.Errorf("%q: %q, want %q", c.name, lines, c.want)
		}
		joined := strings.Join(lines, "")
		if strings.Contains(c.name, " ") {
			joined = strings.Join(lines, " ")
		}
		if joined != c.name {
			t.Errorf("%q wraps to %q, which loses or adds characters", c.name, lines)
		}
		for _, l := range lines {
			if fyne.MeasureText(l, size, fyne.TextStyle{}).Width > width {
				t.Errorf("%q: the line %q is wider than %v", c.name, l, width)
			}
		}
		if len(c.name) > 30 && len(lines) < 2 {
			t.Errorf("%q did not wrap: %q", c.name, lines)
		}
	}
	if got := wrapName("agent-not-ready-taint-key", width, size, fyne.TextStyle{}); !strings.HasSuffix(got[0], "-") {
		t.Errorf("a key wraps after its separators: %q", got)
	}
}

// TestDrawerLongNameNeverRunsIntoItsValue: in a section of names and values a long name without
// spaces wraps within the name column, so its value starts beside it, not over it.
func TestDrawerLongNameNeverRunsIntoItsValue(t *testing.T) {
	f := kindsFake(t, "configmaps")
	newCMFake(t, f)
	m := f.details[ciliumRef.String()]
	m.Sections = append(m.Sections, model.DetailSection{Title: "Settings", Items: []model.DetailItem{
		{Name: "bpf-lb-source-range-all-types", Value: "false"}, {Name: "Name", Value: "x"}}})
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil })
	settle(w)
	var name *paragraph
	for _, p := range findAll[*paragraph](sh.drawer.details) {
		if p.text == "bpf-lb-source-range-all-types" {
			name = p
		}
	}
	if name == nil {
		t.Fatal("the long name is not a wrapped paragraph")
	}
	nameW := sh.app.Settings().Theme().Size(sizeDrawerLabel)
	if name.Size().Width > nameW || name.MinSize().Height < 2*fyne.MeasureText("M", sh.app.Settings().Theme().Size(theme.SizeNameText), fyne.TextStyle{}).Height {
		t.Errorf("the name is %v, more than the %v column, or on one line", name.Size(), nameW)
	}
}

// TestSecretDrawerKeysAboveValues (F3): a Secret's keys stand above their masked values and Show,
// which take the full width; revealing still works (TestSecretRevealInDrawer).
func TestSecretDrawerKeysAboveValues(t *testing.T) {
	f, _ := secretsFake(t)
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the drawer", func() bool { return sh.drawer.last != nil && len(drawerButtons(sh.drawer)["Show"]) == 4 })
	settle(w)
	var key fyne.CanvasObject
	for _, l := range findAll[*selLine](sh.drawer.details) {
		if l.line.text == "DB_PASSWORD" {
			key = l
		}
	}
	show := drawerButtons(sh.drawer)["Show"][2]
	if key == nil {
		t.Fatal("no DB_PASSWORD key")
	}
	drv := fyne.CurrentApp().Driver()
	k, b := drv.AbsolutePositionForObject(key), drv.AbsolutePositionForObject(show)
	if b.Y < k.Y+key.Size().Height-1 {
		t.Errorf("Show at %v is not below its key at %v", b, k)
	}
}

// typeInto replaces a row's value as typing does.
func typeInto(r *cmRow, text string) {
	r.view.SelectAll()
	r.view.ReplaceSelection(text)
}

// toastSays reports whether a shown toast holds s.
func toastSays(sh *shell, s string) bool {
	for _, o := range sh.toasts.Objects {
		for _, l := range findAll[*textLine](o) {
			if strings.Contains(strings.ToLower(l.text), strings.ToLower(s)) {
				return true
			}
		}
		for _, p := range findAll[*paragraph](o) {
			if strings.Contains(strings.ToLower(p.text), strings.ToLower(s)) {
				return true
			}
		}
	}
	return false
}

// renderConfigMapDrawer renders the Config Maps list with cilium-config's drawer: long keys above
// their editors.
func renderConfigMapDrawer(t *testing.T, mode ThemeMode, name string) {
	sh, w, _, _, _ := cmShell(t, mode)
	cd := cmDataOf(sh)
	typeInto(cd.rows["bpf-lb-acceleration"], "native")
	settle(w)
	savePNG(t, w, name)
}

func TestRenderConfigMapDrawerDark(t *testing.T) {
	renderConfigMapDrawer(t, ThemeDark, "configmap_drawer_dark.png")
}

func TestRenderConfigMapDrawerLight(t *testing.T) {
	renderConfigMapDrawer(t, ThemeLight, "configmap_drawer_light.png")
}

// renderConfigMapSave renders the review of a save: the diff of two values and the dry run.
func renderConfigMapSave(t *testing.T, mode ThemeMode, name string) {
	sh, w, q, _, _ := cmShell(t, mode)
	cd := cmDataOf(sh)
	typeInto(cd.rows["allow-localhost"], "always")
	typeInto(cd.rows["bpf-lb-map-max"], "131072")
	test.Tap(cd.save)
	q.until(t, "the review", func() bool { return cd.review != nil && !cd.review.save.Disabled() })
	settle(w)
	savePNG(t, w, name)
}

func TestRenderConfigMapSaveDark(t *testing.T) {
	renderConfigMapSave(t, ThemeDark, "configmap_save_dark.png")
}
func TestRenderConfigMapSaveLight(t *testing.T) {
	renderConfigMapSave(t, ThemeLight, "configmap_save_light.png")
}

// renderConfigMapConflict renders a review that met a newer version of the Config Map.
func TestRenderConfigMapConflictDark(t *testing.T) {
	sh, w, q, _, c := cmShell(t, ThemeDark)
	cd := cmDataOf(sh)
	typeInto(cd.rows["allow-localhost"], "always")
	c.mu.Lock()
	c.stale = "4300"
	c.mu.Unlock()
	test.Tap(cd.save)
	q.until(t, "the stale review", func() bool { return cd.review != nil && cd.review.over == "4300" })
	settle(w)
	savePNG(t, w, "configmap_conflict_dark.png")
}

// TestRenderConfigMapDrawerFooterDark renders the end of the data: an edited value, Revert and
// Save, and the line saying what Save would write.
//
// A Config Map short enough to show whole: Fyne's software painter clips an editor inside the
// drawer's scroll to the editor alone, so one the drawer's top cuts would be drawn over its header
// in the PNG (the window's GL painter clips it to the drawer).
func TestRenderConfigMapDrawerFooterDark(t *testing.T) {
	f := kindsFake(t, "configmaps")
	newCMFakeWith(t, f, map[string]string{
		"agent-not-ready-taint-key": ciliumValues["agent-not-ready-taint-key"], "bpf-lb-acceleration": "disabled",
		"hubble-flowlogs-config.yaml": ciliumValues["hubble-flowlogs-config.yaml"], "kube-proxy-replacement-healthz-bind-address": "",
	})
	sh, w, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the values", func() bool {
		cd := cmDataOf(sh)
		return cd != nil && cd.loaded && len(sh.kinds) > 0 && sh.view.snap != nil && sh.diag.total > 0
	})
	typeInto(cmDataOf(sh).rows["kube-proxy-replacement-healthz-bind-address"], "0.0.0.0:10256")
	settle(w)
	savePNG(t, w, "configmap_drawer_footer_dark.png")
}

// longKeySecretValues are the values of a made-up Secret whose keys are long and have no spaces.
var longKeySecretValues = map[string]string{
	"aws-access-key-id-for-backup-bucket":     "AKIA-not-a-real-key",
	"aws-secret-access-key-for-backup-bucket": "not-a-real-secret-access-key",
	"restic-repository-encryption-password":   "not-a-real-password",
}

// renderSecretLongKeys renders a Secret's drawer whose keys are long: each above its masked value,
// one value revealed.
func renderSecretLongKeys(t *testing.T, mode ThemeMode, name string) {
	f, _ := secretsFake(t)
	data := map[string]any{}
	for k, v := range longKeySecretValues {
		data[k] = base64.StdEncoding.EncodeToString([]byte(v))
	}
	obj := &unstructured.Unstructured{Object: map[string]any{"apiVersion": "v1", "kind": "Secret", "type": "Opaque", "data": data,
		"metadata": map[string]any{"name": "backup-credentials", "namespace": "kube-system", "uid": "uid-backup-credentials",
			"resourceVersion": "51234", "creationTimestamp": ts(40 * 24 * time.Hour)}}}
	ref := model.ObjectRef{Context: "demo", Kind: "secrets", Namespace: "kube-system", Name: "backup-credentials"}
	p, _ := kinds.ByKey("secrets")
	m, err := engine.BuildDetail(p, ref, obj, nil, fixedNow)
	if err != nil {
		t.Fatal(err)
	}
	f.details[ref.String()] = m
	f.handlers[engine.RevealCommand] = func(_ context.Context, inv engine.Invocation) (any, error) {
		var rp engine.RevealParams
		if err := json.Unmarshal(inv.Params, &rp); err != nil {
			return nil, err
		}
		v := longKeySecretValues[rp.Key]
		return engine.SecretValue{Key: rp.Key, Value: v, Size: len(v)}, nil
	}
	f.ui.Selected, f.ui.Drawer = []model.ObjectRef{ref}, &ref
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	q.until(t, "the drawer", func() bool {
		return len(sh.kinds) > 0 && sh.view.snap != nil && sh.drawer.last != nil && len(drawerButtons(sh.drawer)["Show"]) == 3 && sh.diag.total > 0
	})
	test.Tap(drawerButtons(sh.drawer)["Show"][1]) // aws-secret-access-key-for-backup-bucket
	q.until(t, "the value", func() bool { return drawerHasText(sh.drawer, "not-a-real-secret-access-key") })
	settle(w)
	savePNG(t, w, name)
}

func TestRenderSecretLongKeysDark(t *testing.T) {
	renderSecretLongKeys(t, ThemeDark, "secret_longkeys_dark.png")
}

func TestRenderSecretLongKeysLight(t *testing.T) {
	renderSecretLongKeys(t, ThemeLight, "secret_longkeys_light.png")
}
