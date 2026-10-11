package mcp

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

// tokenBytes is the token's entropy: 32 random bytes, 64 hex characters.
const tokenBytes = 32

// newToken returns a fresh bearer token. Hex keeps it safe to paste inside double quotes in any
// shell.
func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("mcp: generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// defaultTokenPath is <user config dir>/Cogpit/mcp-token.
func defaultTokenPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("mcp: token file: %w", err)
	}
	return filepath.Join(dir, "Cogpit", "mcp-token"), nil
}

// writeTokenFile writes the token atomically, mode 0600, into an owner-only directory: a temporary
// file in the same directory, renamed over the target, so a reader never sees a partial token.
// It refuses a symlink as the file or the directory rather than writing through it. A directory
// other users can access is tightened to 0700 when it is Cogpit's own (ownDir) and refused
// otherwise, so that a caller-chosen path such as a home directory never changes mode behind the
// caller's back. On Windows, where modes do not express access, the user's profile ACLs apply.
func writeTokenFile(path, token string, ownDir bool) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mcp: token directory: %w", err)
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("mcp: token directory: %w", err)
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("mcp: token directory %s is a symlink; refusing to write the token through it", dir)
	case !fi.IsDir():
		return fmt.Errorf("mcp: token directory %s is not a directory", dir)
	case runtime.GOOS != "windows" && fi.Mode().Perm()&0o077 != 0:
		if !ownDir {
			return fmt.Errorf("mcp: token directory %s is accessible by other users (mode %#o); it must be 0700", dir, fi.Mode().Perm())
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("mcp: token directory: %w", err)
		}
	}
	switch fi, err := os.Lstat(path); {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fmt.Errorf("mcp: token file: %w", err)
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("mcp: token file %s is a symlink; refusing to write the token through it", path)
	case !fi.Mode().IsRegular():
		return fmt.Errorf("mcp: token file %s is not a regular file", path)
	}

	f, err := os.CreateTemp(dir, ".mcp-token-*")
	if err != nil {
		return fmt.Errorf("mcp: token file: %w", err)
	}
	tmp := f.Name()
	done := false
	defer func() {
		if !done {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o600); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("mcp: token file: %w", err)
	}
	if _, err := f.WriteString(token + "\n"); err != nil {
		return fmt.Errorf("mcp: token file: %w", err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("mcp: token file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("mcp: token file: %w", err)
	}
	// rename replaces a symlink that appeared since the check instead of following it.
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("mcp: token file: %w", err)
	}
	done = true
	return nil
}

// removeTokenFile removes the token file if it still holds token. Another Cogpit started later
// may have replaced it with its own token, which must survive this one's exit; a symlink is not
// Cogpit's either.
func removeTokenFile(path, token string) error {
	if path == "" || token == "" {
		return nil
	}
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("mcp: remove token file: %w", err)
	}
	if !fi.Mode().IsRegular() {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("mcp: remove token file: %w", err)
	}
	if subtle.ConstantTimeCompare(bytes.TrimSpace(data), []byte(token)) != 1 {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("mcp: remove token file: %w", err)
	}
	return nil
}
