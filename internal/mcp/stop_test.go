package mcp

import (
	"context"
	"net"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestStopEndsTheEndpointAndStartBringsItBack: the user's switch in the status bar. Stop removes
// the token file and the listener goes away; Start serves again, with a new token.
func TestStopEndsTheEndpointAndStartBringsItBack(t *testing.T) {
	f := newFixture(t, nil, Options{})
	endpoint, token := f.start(t)
	path := f.server.TokenPath()
	cs := connectHTTP(t, endpoint, token, "")
	t.Cleanup(func() { _ = cs.Close() })
	withToken := func(tok string) func(*http.Request) {
		return func(r *http.Request) { r.Header.Set("Authorization", "Bearer "+tok) }
	}
	if code, _ := post(t, endpoint, withToken(token)); code != http.StatusOK {
		t.Fatalf("before Stop: status %d", code)
	}

	if err := f.server.Stop(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Errorf("token file still there after Stop: %v", err)
	}
	if f.server.URL() != "" || f.server.ClaudeAddCommand() != "" || f.server.ClientConfigJSON() != "" {
		t.Error("a stopped server still hands out its endpoint or command")
	}
	addr := strings.TrimSuffix(strings.TrimPrefix(endpoint, "http://"), endpointPath)
	if c, err := net.DialTimeout("tcp", addr, time.Second); err == nil {
		_ = c.Close()
		t.Error("the port still accepts connections after Stop")
	}
	if err := f.server.Stop(); err != nil {
		t.Errorf("second Stop: %v", err)
	}

	endpoint2, token2 := f.start(t)
	if token2 == token {
		t.Error("the token did not change on restart")
	}
	if cmd := f.server.ClaudeAddCommand(); !strings.Contains(cmd, token2) || strings.Contains(cmd, token) {
		t.Errorf("the registration command carries the wrong token: %q", cmd)
	}
	if code, _ := post(t, endpoint2, withToken(token)); code == http.StatusOK {
		t.Error("the old token is accepted after restart")
	}
	cs2 := connectHTTP(t, endpoint2, token2, "")
	t.Cleanup(func() { _ = cs2.Close() })
	if _, err := cs2.ListTools(context.Background(), nil); err != nil {
		t.Errorf("a tool listing after restart: %v", err)
	}
	if err := f.server.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.server.Start(context.Background()); err == nil {
		t.Error("Start after Close succeeded")
	}
}

// TestStopEndsPendingConfirmation: switching the server off while an agent's confirmation dialog
// is open ends the call and the command never runs; after a restart the server works again.
func TestStopEndsPendingConfirmation(t *testing.T) {
	confirm := newBlockingConfirmer()
	f := newFixture(t, confirm, Options{})
	endpoint, token := f.start(t)
	cs := connectHTTP(t, endpoint, token, "")
	t.Cleanup(func() { _ = cs.Close() })
	done := make(chan struct{})
	go func() {
		defer close(done)
		_, _ = cs.CallTool(context.Background(), &sdk.CallToolParams{Name: "pods_delete", Arguments: map[string]any{"ref": podRef}})
	}()
	confirm.waitEntered(t)
	if err := f.server.Stop(); err != nil {
		t.Fatal(err)
	}
	confirm.waitEnded(t)
	select {
	case <-done: // the call returned, as an error result or a closed connection
	case <-time.After(5 * time.Second):
		t.Fatal("the agent's call did not end")
	}
	if f.deleted.Load() != 0 {
		t.Error("the command ran after the server was switched off")
	}

	endpoint2, token2 := f.start(t)
	cs2 := connectHTTP(t, endpoint2, token2, "")
	t.Cleanup(func() { _ = cs2.Close() })
	if _, err := cs2.ListTools(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
}
