package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/mcp"
)

// TestMCPControlSwitch: the window's switch stops the server (endpoint file gone, port closed, the
// token kept), starts it again with the same token and port (F6), hands the window the command,
// prints it to the terminal and records the user's switches, but not the start at launch, in the
// audit log.
func TestMCPControlSwitch(t *testing.T) {
	var audit, terminal bytes.Buffer
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := engine.New(engine.Options{Audit: &audit, Logger: logger})
	t.Cleanup(s.Close)
	tokenFile := filepath.Join(t.TempDir(), "Cogpit", "mcp-token")
	srv, err := mcp.New(s, mcp.Options{TokenFile: tokenFile, Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	c := &mcpControl{srv: srv, session: s, logger: logger, out: &terminal}

	ep, err := c.Start(false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(ep.URL, "http://127.0.0.1:") || !strings.Contains(ep.Command, ep.URL) {
		t.Errorf("endpoint %+v", ep)
	}
	if audit.Len() != 0 {
		t.Errorf("the start at launch was audited: %s", audit.String())
	}
	if !strings.Contains(terminal.String(), ep.Command) {
		t.Error("the command was not printed to the terminal")
	}
	addr := strings.TrimSuffix(strings.TrimPrefix(ep.URL, "http://"), "/mcp")

	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(tokenFile), mcp.EndpointFileName)); !os.IsNotExist(err) {
		t.Errorf("endpoint file after Stop: %v", err)
	}
	if _, err := os.Lstat(tokenFile); err != nil {
		t.Errorf("token file after Stop: %v, want it kept", err)
	}
	if conn, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
		_ = conn.Close()
		t.Error("the port still accepts connections")
	}

	ep2, err := c.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	if ep2.Command != ep.Command {
		t.Error("the switch changed the registration: the token and the port are kept (F6)")
	}

	var lines []engine.AuditEntry
	for _, l := range strings.Split(strings.TrimSpace(audit.String()), "\n") {
		var e engine.AuditEntry
		if err := json.Unmarshal([]byte(l), &e); err != nil {
			t.Fatalf("audit line %q: %v", l, err)
		}
		lines = append(lines, e)
	}
	if len(lines) != 2 {
		t.Fatalf("audit lines: %s", audit.String())
	}
	for i, enabled := range []string{`{"enabled":false}`, `{"enabled":true}`} {
		e := lines[i]
		if e.Origin != engine.OriginGUI || e.Command != auditMCPSwitch || string(e.Params) != enabled || !e.OK {
			t.Errorf("audit line %d: %+v (params %s)", i, e, e.Params)
		}
	}
	if strings.Contains(audit.String(), "Bearer") {
		t.Error("a token reached the audit log")
	}
}

// TestMCPControlOffersTheBundledCLI: an app whose bundle carries cogpit-cli hands the window the
// stdio registration of that cogpit-cli, which names no port and no token, and prints it; without
// one the HTTP registration, as before.
func TestMCPControlOffersTheBundledCLI(t *testing.T) {
	var terminal bytes.Buffer
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	s := engine.New(engine.Options{Audit: io.Discard, Logger: logger})
	t.Cleanup(s.Close)
	srv, err := mcp.New(s, mcp.Options{TokenFile: filepath.Join(t.TempDir(), "Cogpit", "mcp-token"), Logger: logger})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = srv.Close() })
	cli := "/Users/a b/Applications/Cogpit.app/Contents/MacOS/cogpit-cli"
	c := &mcpControl{srv: srv, session: s, logger: logger, out: &terminal, cli: cli}

	ep, err := c.Start(false)
	if err != nil {
		t.Fatal(err)
	}
	want := "claude mcp remove --scope user cogpit >/dev/null 2>&1; claude mcp add --scope user cogpit -- '" + cli + "' mcp"
	if !ep.Stdio || ep.Command != want || ep.URL == "" {
		t.Errorf("endpoint %+v, want the stdio command %s", ep, want)
	}
	if strings.Contains(ep.Command, "Bearer") || strings.Contains(ep.Command, ep.URL) {
		t.Errorf("the stdio command names this run's endpoint: %s", ep.Command)
	}
	if !strings.Contains(terminal.String(), want) || !strings.Contains(terminal.String(), "once") {
		t.Errorf("printed %q", terminal.String())
	}
	if err := c.Stop(); err != nil {
		t.Fatal(err)
	}

	c.cli = ""
	ep, err = c.Start(true)
	if err != nil {
		t.Fatal(err)
	}
	if ep.Stdio || !strings.Contains(ep.Command, "--transport http") || !strings.Contains(ep.Command, ep.URL) {
		t.Errorf("without a bundled cogpit-cli: %+v", ep)
	}
}
