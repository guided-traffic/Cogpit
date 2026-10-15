package agent

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/testkit"
	"github.com/guided-traffic/cogpit/internal/testkit/fakellm"
)

// writeFile writes a test file and returns its path.
func writeFile(t *testing.T, dir, name string, data []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// TestCAFileTrustsAPrivateCA (the owner's decision of 2026-10-15): an https endpoint whose
// certificate chains to a private CA fails the connection test's TLS layer with a hint at the CA
// file; with the CA's file the test passes and a turn runs; the file is kept with the connection by
// its path and read again when connecting, so a file that changed counts at once.
func TestCAFileTrustsAPrivateCA(t *testing.T) {
	h := newHarness(t, engine.ChatLimits{})
	ca := testkit.NewTestCA(t, "Cogpit test private CA")
	srv := fakellm.NewTLS(t, ca.Issue(t, "127.0.0.1"))
	conn := engine.ChatConnection{Name: "private CA", Kind: engine.ChatLMStudio, BaseURL: srv.Base, Model: "test-model"}

	steps := h.a.TestConnection(context.Background(), conn, nil, nil)
	if got := states(steps); got != "tcp=pass http=fail auth=notReached tools=notReached" ||
		!strings.Contains(steps[1].Detail, "not trusted") || !strings.Contains(steps[1].Detail, "CA file") {
		t.Fatalf("without the CA file: %s %+v", got, steps)
	}
	t.Logf("without the CA file: %s", steps[1].Detail)

	dir := t.TempDir()
	conn.CAFile = writeFile(t, dir, "ca.pem", ca.PEM())
	srv.Script(fakellm.Reply{ToolCalls: []fakellm.ToolCall{{Name: "get_time", Args: `{}`}}})
	steps = h.a.TestConnection(context.Background(), conn, nil, nil)
	if got := states(steps); got != "tcp=pass http=pass auth=pass tools=pass" || !strings.Contains(steps[1].Detail, "TLS 1.") ||
		steps[1].Warning != "" {
		t.Fatalf("with the CA file: %s %+v", got, steps)
	}

	saved, err := h.a.SaveConnection(conn, nil)
	if err != nil || saved.CAFile != conn.CAFile {
		t.Fatalf("saved %+v, %v", saved, err)
	}
	again := New(Options{Session: h.s, Dir: h.dir, Keys: h.keys})
	if !slices.ContainsFunc(again.conns, func(c engine.ChatConnection) bool { return c.CAFile == conn.CAFile }) {
		t.Error("the CA file's path was not kept with the connection")
	}
	c, err := h.a.NewConversation("test", saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	srv.Script(fakellm.Reply{Text: []string{"Hello over TLS."}})
	if err := h.a.Send(c.ID, engine.ChatUserMessage{Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	if got := h.idle(t, c.ID); got.Items[len(got.Items)-1].Text != "Hello over TLS." {
		t.Fatalf("the turn: %v (%s)", itemKinds(got), notes(got))
	}

	// Another CA in the same file: read when connecting, so the server is no longer trusted.
	other := testkit.NewTestCA(t, "another CA")
	writeFile(t, dir, "ca.pem", other.PEM())
	steps = h.a.TestConnection(context.Background(), saved, nil, nil)
	if got := states(steps); got != "tcp=pass http=fail auth=notReached tools=notReached" {
		t.Errorf("after the file changed: %s %+v", got, steps)
	}
}

// TestCAFileErrorsAreNamed: a CA file that does not exist, is not a file or holds no certificate
// fails the TLS layer and a turn with the file named and why, never its content; a CA file is
// refused with an http base URL and as a relative path.
func TestCAFileErrorsAreNamed(t *testing.T) {
	h := newHarness(t, engine.ChatLimits{})
	ca := testkit.NewTestCA(t, "Cogpit test private CA")
	srv := fakellm.NewTLS(t, ca.Issue(t, "127.0.0.1"))
	dir := t.TempDir()
	const content = "not a certificate SECRET-CONTENT-4711"
	for _, tc := range []struct{ path, want string }{
		{filepath.Join(dir, "missing.pem"), "does not exist"},
		{dir, "is not a file"},
		{writeFile(t, dir, "notes.txt", []byte(content)), "holds no PEM certificate"},
	} {
		conn := engine.ChatConnection{Name: "x", Kind: engine.ChatLMStudio, BaseURL: srv.Base, Model: "test-model", CAFile: tc.path}
		steps := h.a.TestConnection(context.Background(), conn, nil, nil)
		if got := states(steps); got != "tcp=pass http=fail auth=notReached tools=notReached" || !strings.Contains(steps[1].Detail, tc.path) ||
			!strings.Contains(steps[1].Detail, tc.want) || strings.Contains(steps[1].Detail, "SECRET-CONTENT") {
			t.Errorf("%s: %s %+v", tc.path, got, steps)
		}
	}

	conn := engine.ChatConnection{Name: "x", Kind: engine.ChatLMStudio, BaseURL: srv.Base, Model: "test-model", CAFile: filepath.Join(dir, "missing.pem")}
	saved, err := h.a.SaveConnection(conn, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := h.a.NewConversation("test", saved.ID)
	if err := h.a.Send(c.ID, engine.ChatUserMessage{Text: "hi"}); err != nil {
		t.Fatal(err)
	}
	if got := notes(h.idle(t, c.ID)); !strings.Contains(got, "missing.pem does not exist") {
		t.Errorf("the turn's note: %q", got)
	}
	if n := len(srv.Requests()); n != 0 {
		t.Errorf("%d requests went out without the CA file", n)
	}

	plain := conn
	plain.BaseURL = h.llm.Base
	if _, err := h.a.SaveConnection(plain, nil); !errors.Is(err, ErrCAFileScheme) {
		t.Errorf("a CA file over http: %v", err)
	}
	if steps := h.a.TestConnection(context.Background(), plain, nil, nil); steps[0].State != engine.ChatTestFail {
		t.Errorf("the test of a CA file over http: %+v", steps)
	}
	relative := conn
	relative.CAFile = "certs/ca.pem"
	if _, err := h.a.SaveConnection(relative, nil); !errors.Is(err, ErrCAFilePath) {
		t.Errorf("a relative CA file: %v", err)
	}
}

// TestCAFileAddsToTheSystemRoots: the pool a CA file makes is the system's roots with the file's
// certificates added (on macOS the platform verifier's pool, which Verify asks first), never the
// file's alone (Go 1.27's crypto/x509; the analysis at loadCAPool).
func TestCAFileAddsToTheSystemRoots(t *testing.T) {
	ca := testkit.NewTestCA(t, "Cogpit test private CA")
	path := writeFile(t, t.TempDir(), "ca.pem", ca.PEM())
	got, _, err := loadCAPool(path)
	if err != nil {
		t.Fatal(err)
	}
	want, err := x509.SystemCertPool()
	if err != nil {
		t.Skipf("no system roots here: %v", err)
	}
	empty := !want.Equal(x509.NewCertPool())
	want.AppendCertsFromPEM(ca.PEM())
	if !got.Equal(want) {
		t.Error("the pool is not the system's roots plus the CA file's certificates")
	}
	alone := x509.NewCertPool()
	alone.AppendCertsFromPEM(ca.PEM())
	if empty && got.Equal(alone) {
		t.Error("the CA file replaced the system's roots")
	}
}

// TestPlainHTTPToAnotherMachineIsWarned (the owner's decision of 2026-10-15, H-95): the connection
// test of an http endpoint on another machine passes its HTTP layer with the warning that the key,
// the conversation and every secret the agent reads travel unencrypted; on this computer
// (localhost, 127.0.0.1, ::1) and over https it does not warn.
func TestPlainHTTPToAnotherMachineIsWarned(t *testing.T) {
	h := newHarness(t, engine.ChatLimits{})
	fake, err := url.Parse(h.llm.Root)
	if err != nil {
		t.Fatal(err)
	}
	// Every dial, the TCP layer's and the client's, reaches the fake endpoint on this computer.
	toFake := func(ctx context.Context, network, _ string) (net.Conn, error) {
		return (&net.Dialer{}).DialContext(ctx, network, fake.Host)
	}
	old := dialTCP
	dialTCP = toFake
	t.Cleanup(func() { dialTCP = old })
	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.DialContext = toFake
	a := New(Options{Session: h.s, Keys: NewMemoryKeyStore(), HTTPClient: &http.Client{Transport: tr, Timeout: 10 * time.Second}})

	for base, warn := range map[string]bool{
		"http://llm.lan.example:1234/v1": true, "http://192.168.1.20:1234/v1": true, "http://[fd00::3]:1234/v1": true,
		"http://localhost:1234/v1": false, "http://127.0.0.1:1234/v1": false, "http://[::1]:1234/v1": false,
	} {
		h.llm.Script(fakellm.Reply{ToolCalls: []fakellm.ToolCall{{Name: "get_time", Args: `{}`}}})
		conn := engine.ChatConnection{Name: "lan", Kind: engine.ChatLMStudio, BaseURL: base, Model: "test-model"}
		steps := a.TestConnection(context.Background(), conn, nil, nil)
		if got := states(steps); got != "tcp=pass http=pass auth=pass tools=pass" {
			t.Fatalf("%s: %s %+v", base, got, steps)
		}
		if warned := steps[1].Warning == engine.ChatPlainHTTPWarning; warned != warn ||
			strings.Contains(steps[1].Detail, "another machine") != warn {
			t.Errorf("%s: warning %q, detail %q; want warned %v", base, steps[1].Warning, steps[1].Detail, warn)
		}
		for i, s := range steps {
			if i != 1 && s.Warning != "" {
				t.Errorf("%s: layer %s warns %q", base, s.Layer, s.Warning)
			}
		}
	}
}
