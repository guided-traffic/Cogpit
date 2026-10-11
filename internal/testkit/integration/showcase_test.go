package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/mcp"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// memConnector connects every context to one in-memory cluster, through the real cache.
type memConnector struct{ src *testkit.MemorySource }

func (m memConnector) Contexts() ([]engine.ContextInfo, error) {
	return []engine.ContextInfo{{Name: "kind-test"}}, nil
}

func (m memConnector) Connect(_ context.Context, name string) (*engine.Backend, error) {
	c := cache.New(m.src, cache.Options{WarmPeriod: time.Minute})
	return &engine.Backend{Context: name, Namespace: "default", Cache: c, Profiles: kinds.Builtin(), Close: c.Close}, nil
}

// syncBuffer is an io.Writer safe for the audit log's concurrent writes.
type syncBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (s *syncBuffer) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.Write(p)
}

func (s *syncBuffer) String() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.b.String()
}

type bearer struct {
	token string
	base  http.RoundTripper
}

func (b bearer) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+b.token)
	return b.base.RoundTrip(r)
}

// TestShowcaseServiceByClusterIP is the MVP's agent showcase end to end through the real
// packages: an MCP client over loopback HTTP asks for "the Service with ClusterIP
// 10.96.100.100"; resources_find finds it in the cache, ui_navigate opens Services, selects it and
// opens its drawer, and the next table snapshot contains its row. Repeated on a warm cache, the
// whole exchange costs no list and no watch (budget P11).
func TestShowcaseServiceByClusterIP(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	src := testkit.NewMemorySource()
	for _, svc := range []struct{ ns, name, ip string }{
		{"cogpit-demo", "web", "10.96.100.100"},
		{"cogpit-demo", "api", "10.96.100.101"},
		{"default", "kubernetes", "10.96.0.1"},
	} {
		src.Set(testkit.ServicesGVR, testkit.Service(svc.ns, svc.name, svc.ip))
	}
	src.Set(testkit.PodsGVR, testkit.Pod("cogpit-demo", "web-0"))

	audit := &syncBuffer{}
	s := engine.New(engine.Options{Connector: memConnector{src: src}, Audit: audit})
	defer s.Close()
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	if err := s.Connect(ctx, "kind-test"); err != nil {
		t.Fatal(err)
	}

	tokenDir := t.TempDir()
	if err := os.Chmod(tokenDir, 0o700); err != nil { // the server refuses a directory others can read
		t.Fatal(err)
	}
	srv, err := mcp.New(s, mcp.Options{TokenFile: filepath.Join(tokenDir, "mcp-token")})
	if err != nil {
		t.Fatal(err)
	}
	if err := srv.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = srv.Close() }()
	token, err := os.ReadFile(srv.TokenPath())
	if err != nil {
		t.Fatal(err)
	}
	tr := &http.Transport{}
	defer tr.CloseIdleConnections()
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "showcase", Version: "1"}, nil).Connect(ctx,
		&sdk.StreamableClientTransport{
			Endpoint:   srv.URL(),
			HTTPClient: &http.Client{Transport: bearer{token: strings.TrimSpace(string(token)), base: tr}},
			MaxRetries: -1,
		}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cs.Close() }()

	want := model.ObjectRef{Context: "kind-test", Kind: "services", Namespace: "cogpit-demo", Name: "web"}
	showcase := func() {
		t.Helper()
		var found commands.FindResult
		callTool(ctx, t, cs, "resources_find", map[string]any{"kind": "services", "field": "clusterIP", "value": "10.96.100.100"}, &found)
		if found.Count != 1 || !found.Matches[0].Ref.Same(want) {
			t.Fatalf("resources_find: %+v", found)
		}
		var nav commands.NavigateResult
		callTool(ctx, t, cs, "ui_navigate", map[string]any{"ref": found.Matches[0].Ref}, &nav)
		if !nav.Found || nav.View != "services" {
			t.Fatalf("ui_navigate: %+v", nav)
		}
		ui := s.UIState()
		if ui.View != "services" || ui.Drawer == nil || !ui.Drawer.Same(want) || len(ui.Selected) != 1 || !ui.Selected[0].Same(want) {
			t.Fatalf("UI state after navigate: view=%q drawer=%v selected=%v", ui.View, ui.Drawer, ui.Selected)
		}
		if !testkit.WaitFor(ctx, func() bool { return s.Table("services").IndexOf(want) >= 0 }) {
			t.Fatal("the Services snapshot never contained the row")
		}
		var state commands.StateResult
		callTool(ctx, t, cs, "ui_state", map[string]any{}, &state)
	}

	showcase()
	lists, watches := src.Lists(), src.Watches()
	// Leave the view so the informer is released, then repeat on the warm cache.
	callTool(ctx, t, cs, "ui_openView", map[string]any{"view": "pods"}, nil)
	showcase()
	if src.Lists() != lists || src.Watches() != watches {
		t.Errorf("warm repeat cost %d lists and %d watches, want 0 and 0 (P11)", src.Lists()-lists, src.Watches()-watches)
	}
	if !strings.Contains(audit.String(), `"origin":"externalAgent","command":"ui.navigate"`) {
		t.Errorf("ui.navigate from the agent is not in the audit log:\n%s", audit.String())
	}
}

func callTool(ctx context.Context, t *testing.T, cs *sdk.ClientSession, name string, args map[string]any, out any) {
	t.Helper()
	res, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	var text string
	for _, c := range res.Content {
		if tc, ok := c.(*sdk.TextContent); ok {
			text += tc.Text
		}
	}
	if res.IsError {
		t.Fatalf("%s failed: %s", name, text)
	}
	if out != nil {
		if err := json.Unmarshal([]byte(text), out); err != nil {
			t.Fatalf("%s: decode %q: %v", name, text, err)
		}
	}
}
