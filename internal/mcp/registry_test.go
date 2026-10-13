package mcp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
	"github.com/guided-traffic/cogpit/internal/model"
)

// realFixture is the fixture over Cogpit's own registry (package commands), without a cluster.
func realFixture(t *testing.T, confirmer engine.Confirmer) *fixture {
	t.Helper()
	f := &fixture{audit: &syncBuffer{}, logs: &syncBuffer{}}
	f.session = engine.New(engine.Options{Confirmer: confirmer, Audit: f.audit})
	t.Cleanup(f.session.Close)
	if err := commands.RegisterAll(f.session.Registry()); err != nil {
		t.Fatal(err)
	}
	srv, err := New(f.session, Options{TokenFile: filepath.Join(t.TempDir(), "Cogpit", "mcp-token")})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	f.server = srv
	return f
}

// sampleArgs returns arguments that decode for a command: every required property with a value of
// its type, a ref as a Secret's reference.
func sampleArgs(cmd *engine.Command) map[string]any {
	args := map[string]any{}
	if cmd.Schema == nil {
		return args
	}
	for _, name := range cmd.Schema.Required {
		args[name] = sampleArg(name, cmd.Schema.Properties[name])
	}
	return args
}

func sampleArg(name string, s *jsonschema.Schema) any {
	if name == engine.ParamRef {
		return model.ObjectRef{Context: "test", Kind: "secrets", Namespace: "shop", Name: "web"}
	}
	if s != nil && len(s.Enum) > 0 {
		return s.Enum[0]
	}
	switch engine.SchemaType(s) {
	case "integer", "number":
		return 1
	case "boolean":
		return true
	case "array":
		return []any{}
	case "object":
		return map[string]any{}
	}
	return "x"
}

// TestEveryCommandIsATool: Cogpit's whole registry, the actions other scopes add included, is
// served as MCP tools, one per command with the command's schema; a GUI-only command's tool says
// so, and calling it is refused with its reason (code guiOnly) before anything is asked, and
// audited.
func TestEveryCommandIsATool(t *testing.T) {
	var asked atomic.Int32
	f := realFixture(t, engine.ConfirmFunc(func(context.Context, engine.ConfirmRequest) (bool, error) {
		asked.Add(1)
		return false, nil
	}))
	cs := f.connect(t)
	tools := map[string]*sdk.Tool{}
	for tool, err := range cs.Tools(context.Background(), nil) {
		if err != nil {
			t.Fatal(err)
		}
		tools[tool.Name] = tool
	}
	cmds := f.session.Registry().Commands()
	if len(tools) != len(cmds) {
		t.Errorf("%d tools for %d commands", len(tools), len(cmds))
	}
	guiOnly := 0
	for _, cmd := range cmds {
		tool := tools[cmd.ToolName()]
		if tool == nil {
			t.Errorf("command %s has no tool", cmd.ID)
			continue
		}
		var want any
		data, _ := json.Marshal(cmd.Schema)
		if err := json.Unmarshal(data, &want); err != nil {
			t.Fatal(err)
		}
		if root, ok := want.(map[string]any); ok && root["type"] == nil { // a pointer's ["null","object"] is served as "object"
			root["type"], root["types"] = "object", nil
			delete(root, "types")
		}
		if !reflect.DeepEqual(tool.InputSchema, want) {
			t.Errorf("%s: input schema %v, want %v", tool.Name, tool.InputSchema, want)
		}
		if !cmd.GUIOnly {
			continue
		}
		guiOnly++
		if !strings.Contains(tool.Description, "GUI only: ") {
			t.Errorf("%s: the description does not say it is GUI-only: %q", tool.Name, tool.Description)
		}
		res := call(t, cs, tool.Name, sampleArgs(cmd))
		if !res.IsError || errorCode(res) != codeGUIOnly || !strings.Contains(text(res), "Cogpit's") || !strings.Contains(text(res), "window") {
			t.Errorf("%s: isError %v, code %q, text %q; want the GUI-only refusal", tool.Name, res.IsError, errorCode(res), text(res))
		}
	}
	if guiOnly == 0 {
		t.Error("no GUI-only command in the registry")
	}
	if n := asked.Load(); n != 0 {
		t.Errorf("the user was asked %d times for GUI-only commands", n)
	}
	refused := 0
	for _, e := range f.auditEntries(t) {
		if e.Origin == engine.OriginExternalAgent && !e.OK && strings.Contains(e.Error, "window") {
			refused++
		}
	}
	if refused != guiOnly {
		t.Errorf("%d refusals in the audit log, want %d", refused, guiOnly)
	}
}

// screenWindow is a window that shows the same lines in every tab.
type screenWindow []string

func (w screenWindow) Len() int           { return len(w) }
func (w screenWindow) Number(i int) int64 { return int64(i + 1) }
func (w screenWindow) Text(i int) string  { return w[i] }

func (w screenWindow) ReadDockOutput(_ context.Context, _ engine.DockTab, q engine.DockOutputQuery) (engine.DockOutput, error) {
	return engine.DockOutput{DockPage: engine.PageDockLines(w, q), Shown: len(w)}, nil
}

// TestDockReadOverMCP (D-44): over the server, an agent reads a logs tab's lines, and a terminal
// tab is refused with the reason that names the user's switch, without a line of it; the switch
// itself is refused as GUI-only.
func TestDockReadOverMCP(t *testing.T) {
	f := realFixture(t, nil)
	f.session.SetDockOutputReader(screenWindow{"$ export TOKEN=s3cr3t", "GET /healthz 200"})
	f.session.UpdateUI(func(u *engine.UIState) {
		u.Dock = append(u.Dock,
			engine.DockTab{ID: "logs-1", Type: engine.DockLogs, Title: "Logs: web-0/app"},
			engine.DockTab{ID: "terminal-1", Type: engine.DockTerminal, Title: "Terminal 1"})
	})
	cs := f.connect(t)

	res := call(t, cs, "dock_read", map[string]any{"id": "logs-1", "limit": 1})
	var out engine.DockOutput
	decode(t, res.StructuredContent, &out)
	if res.IsError || len(out.Lines) != 1 || out.Lines[0] != "GET /healthz 200" || !out.More || out.Tab.ID != "logs-1" {
		t.Fatalf("logs tab: isError %v, %+v", res.IsError, out)
	}

	res = call(t, cs, "dock_read", map[string]any{"id": "terminal-1"})
	if !res.IsError || !strings.Contains(text(res), "Agents may read") || strings.Contains(text(res), "s3cr3t") {
		t.Errorf("terminal tab: isError %v, %q", res.IsError, text(res))
	}
	res = call(t, cs, "dock_allowAgentRead", map[string]any{"id": "terminal-1", "allowed": true})
	if !res.IsError || errorCode(res) != codeGUIOnly {
		t.Errorf("the switch from an agent: isError %v, code %q, %q", res.IsError, errorCode(res), text(res))
	}
	if tab, _ := f.session.UIState().DockTab("terminal-1"); tab.AgentRead {
		t.Error("an agent turned the switch on")
	}
}
