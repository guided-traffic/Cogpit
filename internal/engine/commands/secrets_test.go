package commands_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// secretsServer is the cluster side of the Secrets drawer: the full Secrets of the source the
// cache watches as metadata, read by the drawer's GET, which it counts.
type secretsServer struct {
	*fakeWriter
	src *testkit.MemorySource

	mu   sync.Mutex
	gets int
}

var _ engine.SecretReader = (*secretsServer)(nil)

func (w *secretsServer) GetSecret(_ context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	w.mu.Lock()
	w.gets++
	w.mu.Unlock()
	list, err := w.src.ListerWatcher(cache.Scope{GVR: testkit.SecretsGVR}).List(metav1.ListOptions{}) //nolint:staticcheck // as objectServer.current
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

// secretsConnector connects "test" with the real Secrets profile and the secretsServer.
type secretsConnector struct {
	src    *testkit.MemorySource
	server *secretsServer
}

func (c *secretsConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: "test", Current: true}}, nil
}

func (c *secretsConnector) Connect(_ context.Context, name string) (*engine.Backend, error) {
	ch := cache.New(c.src, cache.Options{WarmPeriod: time.Minute})
	p, _ := kinds.ByKey("secrets")
	return &engine.Backend{
		Context: name, Namespace: "default", Cache: ch, Close: ch.Close,
		Profiles: append([]*model.KindProfile{p}, testProfiles()...),
		Conn:     &cluster.Conn{}, Actions: c.server,
	}, nil
}

// secretsHarness is a session on the Secrets connector with an audit log and a log, recording
// every event.
type secretsHarness struct {
	s      *engine.Session
	server *secretsServer
	audit  *syncBuffer
	logs   *bytes.Buffer
	mu     sync.Mutex
	events []engine.Event
}

func newSecretsHarness(t *testing.T) *secretsHarness {
	t.Helper()
	src := testkit.NewMemorySource()
	src.Set(testkit.SecretsGVR, testkit.Secret("apps", "db", map[string]string{"app": "db"},
		map[string]string{"password": "hunter2", "dsn": "postgres://app:hunter2@db:5432/app"}))
	h := &secretsHarness{server: &secretsServer{fakeWriter: &fakeWriter{}, src: src}, audit: &syncBuffer{}, logs: &bytes.Buffer{}}
	var logMu sync.Mutex
	h.s = engine.New(engine.Options{
		Connector: &secretsConnector{src: src, server: h.server}, SnapshotInterval: 10 * time.Millisecond,
		Audit:  h.audit,
		Logger: slog.New(slog.NewTextHandler(lockedWriter{&logMu, h.logs}, &slog.HandlerOptions{Level: slog.LevelDebug})),
	})
	if err := commands.RegisterAll(h.s.Registry()); err != nil {
		t.Fatal(err)
	}
	h.s.Subscribe(func(ev engine.Event) {
		h.mu.Lock()
		h.events = append(h.events, ev)
		h.mu.Unlock()
	})
	t.Cleanup(h.s.Close)
	if err := h.s.Connect(context.Background(), "test"); err != nil {
		t.Fatal(err)
	}
	return h
}

type lockedWriter struct {
	mu *sync.Mutex
	w  *bytes.Buffer
}

func (l lockedWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

func (h *secretsHarness) dispatch(origin engine.Origin, id string, params any) (any, error) {
	raw, _ := json.Marshal(params)
	return h.s.Dispatch(context.Background(), engine.Invocation{ID: id, Params: raw, Origin: origin})
}

// The drawer of a Secret lists its keys masked, and secret.reveal hands one value to the GUI and
// to nobody else: agents and the CLI are refused; no value reaches object_get, the audit log, a
// toast or the log.
func TestRevealIsGUIOnly(t *testing.T) {
	h := newSecretsHarness(t)
	ref := model.ObjectRef{Context: "test", Kind: "secrets", Namespace: "apps", Name: "db"}
	reveal := engine.RevealParams{Ref: ref, Key: "password"}

	// Before the drawer opened, nothing is held to reveal.
	if _, err := h.dispatch(engine.OriginGUI, "secret.reveal", reveal); err == nil || !strings.Contains(err.Error(), "drawer is open") {
		t.Errorf("reveal without the drawer: %v", err)
	}

	// An agent opens the drawer (ui.navigate) and reads it: keys, masked.
	if _, err := h.dispatch(engine.OriginExternalAgent, "ui.navigate", map[string]any{"ref": ref}); err != nil {
		t.Fatal(err)
	}
	var m engine.DetailModel
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if !testkit.WaitFor(ctx, func() bool {
		res, err := h.dispatch(engine.OriginExternalAgent, "object.get", map[string]any{"ref": ref})
		if err != nil {
			return false
		}
		b, _ := json.Marshal(res)
		m = engine.DetailModel{}
		_ = json.Unmarshal(b, &m)
		return strings.Contains(string(b), engine.MaskedValue)
	}) {
		t.Fatalf("object.get never showed the masked data: %+v", m)
	}
	if b, _ := json.Marshal(m); strings.Contains(string(b), "hunter2") {
		t.Fatalf("object.get for an agent: %s", b)
	}

	for _, origin := range []engine.Origin{engine.OriginExternalAgent, engine.OriginBuiltInAgent, engine.OriginCLI} {
		res, err := h.dispatch(origin, "secret.reveal", reveal)
		if err == nil || res != nil || !strings.Contains(err.Error(), "never receive them") {
			t.Errorf("%s: reveal gave %v, %v", origin, res, err)
		}
	}
	res, err := h.dispatch(engine.OriginGUI, "secret.reveal", reveal)
	if err != nil {
		t.Fatal(err)
	}
	if v := res.(engine.SecretValue); v.Value != "hunter2" || v.Key != "password" || v.Size != 7 {
		t.Errorf("GUI reveal %+v", v)
	}
	if _, err := h.dispatch(engine.OriginGUI, "secret.reveal", engine.RevealParams{Ref: ref, Key: "nope"}); err == nil {
		t.Error("a key the Secret does not have was revealed")
	}
	if h.server.gets != 1 {
		t.Errorf("%d GETs; the reveals must not read the cluster", h.server.gets)
	}

	// Nothing of a value anywhere else.
	var trail bytes.Buffer
	for _, e := range h.audit.Lines() {
		b, _ := json.Marshal(e)
		trail.Write(b)
	}
	h.mu.Lock()
	for _, ev := range h.events {
		b, _ := json.Marshal(ev)
		trail.Write(b)
	}
	h.mu.Unlock()
	trail.WriteString(h.logs.String())
	for _, leak := range []string{"hunter2", "aHVudGVyMg==", "5432"} {
		if strings.Contains(trail.String(), leak) {
			t.Errorf("%q in the audit log, the events or the log", leak)
		}
	}
	refused := 0
	for _, e := range h.audit.Lines() {
		if e.Command == "secret.reveal" && !e.OK {
			refused++
		}
	}
	if refused != 2 {
		t.Errorf("%d refused agent reveals audited, want 2 (the CLI is not audited)", refused)
	}

	// Closing the drawer drops the data: the GUI cannot reveal it either.
	if _, err := h.dispatch(engine.OriginGUI, "ui.closeDrawer", struct{}{}); err != nil {
		t.Fatal(err)
	}
	if _, err := h.dispatch(engine.OriginGUI, "secret.reveal", reveal); err == nil {
		t.Error("revealed after the drawer closed")
	}
}

// Secrets stay out of Edit and Create from YAML, also with the metadata-only view: Edit is
// disabled, Create refuses a Secret's manifest.
func TestSecretsStayRefusedForEditAndCreate(t *testing.T) {
	h := newSecretsHarness(t)
	ref := model.ObjectRef{Context: "test", Kind: "secrets", Namespace: "apps", Name: "db"}
	if _, err := h.s.Objects(context.Background(), "secrets", nil); err != nil {
		t.Fatal(err)
	}
	cmd, _ := h.s.Registry().Get("object.edit")
	raw, _ := json.Marshal(map[string]any{"ref": ref})
	if err := cmd.Enabled(h.s, raw); err == nil || !strings.Contains(err.Error(), "cannot be edited") {
		t.Errorf("object.edit on a Secret: %v", err)
	}
	_, err := h.dispatch(engine.OriginGUI, "object.create", map[string]any{
		"yaml": "apiVersion: v1\nkind: Secret\nmetadata:\n  name: new\n  namespace: apps\nstringData:\n  k: v\n",
	})
	if err == nil || !strings.Contains(err.Error(), "values are redacted") {
		t.Errorf("object.create of a Secret: %v", err)
	}
}
