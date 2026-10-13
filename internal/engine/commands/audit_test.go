package commands_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// auditHarness is the command harness with its audit log in a file that audit.list reads back.
func auditHarness(t *testing.T) (*harness, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "audit.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Close() })
	return newHarness(t, seed(), engine.Options{Audit: f, AuditPath: path}), path
}

// audit.list reads the audit log newest first: what agents dispatched and the user's switch of the
// MCP server, each with its origin, context, phase, outcome and the objects its parameters name;
// filtered by origin, paged; the user's own commands and the CLI's are not in it; a Secret an agent
// sent never shows its value; each new line is announced (EventAudit) so that the window reads
// again.
func TestAuditListNewestFirstAndBounded(t *testing.T) {
	h, _ := auditHarness(t)
	agent, cli, gui := engine.OriginExternalAgent, engine.OriginCLI, engine.OriginGUI
	announced := make(chan struct{}, 16)
	cancel := h.s.Subscribe(func(ev engine.Event) {
		if ev.Type == engine.EventAudit {
			select {
			case announced <- struct{}{}:
			default:
			}
		}
	})
	defer cancel()

	h.must(t, agent, "ui.openView", map[string]any{"view": "pods"}, nil)
	h.must(t, cli, "kinds.list", struct{}{}, nil)                             // the CLI's commands are not audited
	h.s.AuditUser("mcp.switch", map[string]bool{"enabled": false}, nil)       // the user's switch of the MCP server
	if _, err := h.s.Objects(context.Background(), "pods", nil); err != nil { // the delete's Enabled reads the cache
		t.Fatal(err)
	}
	pod := model.ObjectRef{Context: "test", Kind: "pods", Namespace: "shop", Name: "web-0"}
	if err := h.run(t, agent, "object.delete", map[string]any{"ref": pod}, nil); !errors.Is(err, engine.ErrConfirmationUnavailable) {
		t.Fatalf("a headless agent's delete: %v", err)
	}
	secret := "apiVersion: v1\nkind: Secret\nmetadata: {name: db, namespace: shop}\nstringData: {password: audit-HUNTER2-value}\n"
	_ = h.run(t, engine.OriginBuiltInAgent, "object.create", map[string]any{"yaml": secret}, nil) // refused: no matter
	h.must(t, gui, "ui.setFilter", map[string]any{"filter": "web"}, nil)                          // the user's own: not audited
	select {
	case <-announced:
	case <-time.After(5 * time.Second):
		t.Fatal("no EventAudit after the agents' commands")
	}

	var all engine.AuditList
	h.must(t, gui, "audit.list", struct{}{}, &all)
	got := make([]string, len(all.Entries))
	for i, e := range all.Entries {
		got[i] = string(e.Origin) + " " + e.Command
		if i > 0 && e.Time.After(all.Entries[i-1].Time) {
			t.Errorf("entry %d (%s) is newer than the one before", i, got[i])
		}
		if e.Origin == agent && e.Context != "test" {
			t.Errorf("%s: context %q", got[i], e.Context)
		}
	}
	want := []string{"builtInAgent object.create", "externalAgent object.delete", "gui mcp.switch", "externalAgent ui.openView"}
	if strings.Join(got, ", ") != strings.Join(want, ", ") || all.Matched != 4 || all.Returned != 4 || all.Partial || all.Skipped != 0 {
		t.Fatalf("entries %v (matched %d, returned %d, partial %v, skipped %d), want %v", got, all.Matched, all.Returned, all.Partial, all.Skipped, want)
	}
	del := all.Entries[1]
	if del.OK || del.Phase != "done" || !strings.Contains(del.Error, "confirmation") || len(del.Objects) != 1 || !del.Objects[0].Same(pod) || del.Label != "Delete" {
		t.Errorf("the refused delete: %+v", del)
	}
	b, _ := json.Marshal(all)
	if strings.Contains(string(b), "HUNTER2") {
		t.Fatalf("the audit list carries the Secret's value: %s", b)
	}

	list := func(origin engine.Origin, params map[string]any) engine.AuditList {
		t.Helper()
		var page engine.AuditList
		h.must(t, origin, "audit.list", params, &page)
		return page
	}
	if page := list(agent, map[string]any{"origin": "agents"}); page.Matched != 3 || page.Entries[0].Command != "object.create" { // an agent reads it too
		t.Errorf("agents only: %+v", page)
	}
	if page := list(gui, map[string]any{"origin": "gui"}); page.Matched != 1 || page.Entries[0].Command != "mcp.switch" || page.Entries[0].Label != "" {
		t.Errorf("the window's only: %+v", page)
	}
	if page := list(gui, map[string]any{"origin": "cli"}); page.Matched != 0 || len(page.Entries) != 0 {
		t.Errorf("cli only: %+v", page)
	}
	// The agent's own audit.list above is the newest entry now; the page skips it.
	if page := list(gui, map[string]any{"offset": 1, "limit": 2}); page.Matched != 5 || page.Returned != 2 || page.Offset != 1 ||
		page.Entries[0].Command != "object.create" || page.Entries[1].Command != "object.delete" {
		t.Errorf("offset 1, limit 2: %+v", page)
	}
	if err := h.run(t, gui, "audit.list", map[string]any{"origin": "robots"}, nil); err == nil {
		t.Error("an unknown origin was accepted")
	}
}

// The audit log is read from its end, at most its last 4 MiB, from a line's start: a longer file
// lists its newest entries and says it is partial; a page has at most 1000; a line that is no entry
// is counted, not listed; a Secret in a line an older Cogpit wrote is redacted on the way out.
func TestAuditListReadsTheEndOfALongLog(t *testing.T) {
	h, path := auditHarness(t)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	pad := strings.Repeat("x", 150)
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	const n = 30000 // about 7 MB
	for i := range n {
		line, _ := json.Marshal(engine.AuditEntry{Time: start.Add(time.Duration(i) * time.Second), Origin: engine.OriginExternalAgent,
			Command: "ui.setFilter", Params: json.RawMessage(fmt.Sprintf(`{"filter":"%s-%05d"}`, pad, i)), Phase: "done", OK: true})
		_, _ = f.Write(append(line, '\n'))
	}
	_, _ = f.WriteString("not an entry\n")
	old, _ := json.Marshal(engine.AuditEntry{Time: start.Add(n * time.Second), Origin: engine.OriginExternalAgent, Command: "object.apply",
		Params: json.RawMessage(`{"yaml":"apiVersion: v1\nkind: Secret\nmetadata: {name: s}\ndata: {k: c2VjcmV0LU9MRC12YWx1ZQ==}\n"}`), Phase: "done"})
	_, _ = f.Write(append(old, '\n'))
	_ = f.Close()

	var got engine.AuditList
	h.must(t, engine.OriginGUI, "audit.list", map[string]any{"limit": 5000}, &got)
	if !got.Partial || got.Returned != engine.MaxAuditEntries || got.Matched >= n || got.Matched < 10000 || got.Skipped != 1 {
		t.Fatalf("partial %v, returned %d, matched %d of %d, skipped %d", got.Partial, got.Returned, got.Matched, n, got.Skipped)
	}
	if got.Entries[0].Command != "object.apply" || !strings.Contains(got.Entries[1].Params.String(), fmt.Sprintf("-%05d", n-1)) {
		t.Errorf("not newest first: %s, %s", got.Entries[0].Command, got.Entries[1].Params)
	}
	if b, _ := json.Marshal(got.Entries[0]); strings.Contains(string(b), "c2VjcmV0LU9MRC12YWx1ZQ") {
		t.Errorf("an old line's Secret value is listed: %s", b)
	}
}

// audit.open opens one Audit Log dock tab, or activates it with the filter asked for; agents may
// open it (a UI command); without an audit log file it is disabled and says why.
func TestAuditOpenTab(t *testing.T) {
	h, _ := auditHarness(t)
	var first, again struct {
		Tab     engine.DockTab `json:"tab"`
		Created bool           `json:"created"`
	}
	h.must(t, engine.OriginGUI, "audit.open", struct{}{}, &first)
	if !first.Created || first.Tab.Type != engine.DockAudit || first.Tab.Title != "Audit Log" || first.Tab.Audit.Origin != "" {
		t.Fatalf("the tab: %+v", first)
	}
	h.must(t, engine.OriginGUI, "terminal.open", struct{}{}, nil)
	h.must(t, engine.OriginExternalAgent, "audit.open", map[string]any{"origin": "agents"}, &again)
	u := h.s.UIState()
	n := 0
	for _, tab := range u.Dock {
		if tab.Type == engine.DockAudit {
			n++
		}
	}
	if again.Created || again.Tab.ID != first.Tab.ID || again.Tab.Audit.Origin != "agents" || u.ActiveDock != first.Tab.ID || n != 1 {
		t.Errorf("opened again: %+v; active %s; %d audit tabs", again, u.ActiveDock, n)
	}

	plain := newHarness(t, seed(), engine.Options{})
	err := plain.run(t, engine.OriginGUI, "audit.open", struct{}{}, nil)
	var dis *engine.DisabledError
	if !errors.As(err, &dis) || !strings.Contains(dis.Reason, "no audit log") {
		t.Errorf("audit.open without a file: %v", err)
	}
}
