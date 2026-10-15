package mcp

import (
	"context"
	"encoding/hex"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// privateDir returns a new owner-only directory; t.TempDir's own directories are 0755.
func privateDir(t testing.TB) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "private")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	return dir
}

func mode(t testing.TB, path string) os.FileMode {
	t.Helper()
	fi, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	return fi.Mode().Perm()
}

func TestTokenFileLifecycle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "Cogpit")
	path := filepath.Join(dir, "mcp-token")
	f := newFixture(t, nil, Options{TokenFile: path})
	if f.server.TokenPath() != path {
		t.Errorf("TokenPath %q before Start", f.server.TokenPath())
	}
	_, token := f.start(t)
	if b, err := hex.DecodeString(token); err != nil || len(b) != tokenBytes {
		t.Errorf("token is not %d hex-encoded bytes", tokenBytes)
	}
	if runtime.GOOS != "windows" {
		if m := mode(t, path); m != 0o600 {
			t.Errorf("token file mode %#o, want 0600", m)
		}
		if m := mode(t, dir); m != 0o700 {
			t.Errorf("token directory mode %#o, want 0700", m)
		}
	}
	if leftovers, _ := filepath.Glob(filepath.Join(dir, ".mcp-token-*")); len(leftovers) != 0 {
		t.Errorf("temporary files left: %v", leftovers)
	}
	if err := f.server.Start(context.Background()); err == nil {
		t.Error("second Start succeeded")
	}
	if err := f.server.Close(); err != nil {
		t.Fatal(err)
	}
	// F6: the token stays for the next start; only the user's Regenerate token replaces it.
	if data, err := os.ReadFile(path); err != nil || strings.TrimSpace(string(data)) != token {
		t.Errorf("token file after Close: %v (holds the token: %v)", err, strings.TrimSpace(string(data)) == token)
	}
	if err := f.server.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
	if err := f.server.Start(context.Background()); err == nil {
		t.Error("Start after Close succeeded")
	}
}

// TestTokenKeptAcrossStarts (F6): a later start, of the same server or of the next Cogpit, serves
// the token the token file keeps, so that a client registered once keeps working.
func TestTokenKeptAcrossStarts(t *testing.T) {
	path := filepath.Join(privateDir(t), "mcp-token")
	first := newFixture(t, nil, Options{TokenFile: path})
	_, token := first.start(t)
	if err := first.server.Close(); err != nil {
		t.Fatal(err)
	}
	next := newFixture(t, nil, Options{TokenFile: path})
	endpoint, again := next.start(t)
	if again != token || next.server.Token() != token {
		t.Error("the next start made a new token")
	}
	cs := connectHTTP(t, endpoint, token, "")
	defer func() { _ = cs.Close() }()
	if _, err := cs.ListTools(context.Background(), nil); err != nil {
		t.Errorf("the kept token is refused: %v", err)
	}
}

// TestTokenFileNotKept: a token file that holds no token Cogpit wrote, or that other users may
// read, is replaced by a new token at the next start (owner-only again); a symlink is refused
// (TestStartRefusesSymlinks).
func TestTokenFileNotKept(t *testing.T) {
	kept := strings.Repeat("ab", tokenBytes)
	cases := map[string]struct {
		content string
		mode    os.FileMode
	}{
		"not a token": {"not-a-token", 0o600},
		"short":       {kept[:32], 0o600},
	}
	if runtime.GOOS != "windows" {
		cases["readable by others"] = struct {
			content string
			mode    os.FileMode
		}{kept, 0o644}
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(privateDir(t), "mcp-token")
			if err := os.WriteFile(path, []byte(c.content+"\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(path, c.mode); err != nil {
				t.Fatal(err)
			}
			f := newFixture(t, nil, Options{TokenFile: path})
			_, token := f.start(t)
			if token == c.content || !validToken(token) {
				t.Error("the start kept a token it should not")
			}
			if runtime.GOOS != "windows" {
				if m := mode(t, path); m != 0o600 {
					t.Errorf("token file mode %#o, want 0600", m)
				}
			}
			if !strings.Contains(f.logs.String(), "MCP token not kept") {
				t.Error("the replacement was not logged")
			}
			if strings.Contains(f.logs.String(), token) || strings.Contains(f.logs.String(), kept) {
				t.Error("a token reached the logs")
			}
		})
	}
}

func TestStartRefusesSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	base := privateDir(t)
	target := filepath.Join(base, "target")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(base, "mcp-token")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	realDir := filepath.Join(base, "real")
	if err := os.Mkdir(realDir, 0o700); err != nil {
		t.Fatal(err)
	}
	linkDir := filepath.Join(base, "linked")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{link, filepath.Join(linkDir, "mcp-token")} {
		f := newFixture(t, nil, Options{TokenFile: path})
		err := f.server.Start(context.Background())
		if err == nil || !strings.Contains(err.Error(), "symlink") {
			t.Errorf("Start with %s: err %v, want a symlink refusal", path, err)
		}
		if f.server.URL() != "" {
			t.Errorf("server running after a refused start")
		}
	}
	if data, _ := os.ReadFile(target); string(data) != "keep" {
		t.Errorf("symlink target changed to %q", data)
	}
	if entries, _ := os.ReadDir(realDir); len(entries) != 0 {
		t.Errorf("wrote through the directory symlink: %v", entries)
	}
}

func TestTokenDirectoryMustBeOwnerOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("modes do not express access on Windows")
	}
	open := filepath.Join(t.TempDir(), "shared")
	if err := os.Mkdir(open, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(open, 0o755); err != nil {
		t.Fatal(err)
	}
	// A caller-chosen directory is never tightened behind the caller's back.
	f := newFixture(t, nil, Options{TokenFile: filepath.Join(open, "mcp-token")})
	if err := f.server.Start(context.Background()); err == nil || !strings.Contains(err.Error(), "accessible by other users") {
		t.Errorf("Start into a 0755 directory: %v", err)
	}
	if m := mode(t, open); m != 0o755 {
		t.Errorf("caller's directory changed to %#o", m)
	}
	// Cogpit's own directory is tightened.
	if err := writeTokenFile(filepath.Join(open, "mcp-token"), "t", true); err != nil {
		t.Fatal(err)
	}
	if m := mode(t, open); m != 0o700 {
		t.Errorf("own directory mode %#o, want 0700", m)
	}
}
