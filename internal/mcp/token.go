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

// goosWindows is runtime.GOOS on Windows, where file modes do not express access: the user's
// profile ACLs apply instead.
const goosWindows = "windows"

// newToken returns a fresh bearer token. Hex keeps it safe to paste inside double quotes in any
// shell.
func newToken() (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("mcp: generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// TokenFileName is the token file's name in Cogpit's configuration directory.
const TokenFileName = "mcp-token"

// defaultTokenPath is <user config dir>/Cogpit/mcp-token.
func defaultTokenPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("mcp: token file: %w", err)
	}
	return filepath.Join(dir, "Cogpit", TokenFileName), nil
}

// writeTokenFile writes the token atomically, mode 0600, into an owner-only directory: a temporary
// file in the same directory, renamed over the target, so a reader never sees a partial token.
// It refuses a symlink as the file or the directory rather than writing through it. A directory
// other users can access is tightened to 0700 when it is Cogpit's own (ownDir) and refused
// otherwise, so that a caller-chosen path such as a home directory never changes mode behind the
// caller's back. On Windows, where modes do not express access, the user's profile ACLs apply.
func writeTokenFile(path, token string, ownDir bool) error {
	return writePrivateLine(path, token, "token", ownDir)
}

// writePrivateLine writes one line to path as writeTokenFile describes; what names the file in
// errors ("token", "endpoint").
func writePrivateLine(path, line, what string, ownDir bool) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("mcp: %s directory: %w", what, err)
	}
	fi, err := os.Lstat(dir)
	if err != nil {
		return fmt.Errorf("mcp: %s directory: %w", what, err)
	}
	switch {
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("mcp: %s directory %s is a symlink; refusing to write it through it", what, dir)
	case !fi.IsDir():
		return fmt.Errorf("mcp: %s directory %s is not a directory", what, dir)
	case runtime.GOOS != goosWindows && fi.Mode().Perm()&0o077 != 0:
		if !ownDir {
			return fmt.Errorf("mcp: %s directory %s is accessible by other users (mode %#o); it must be 0700", what, dir, fi.Mode().Perm())
		}
		if err := os.Chmod(dir, 0o700); err != nil {
			return fmt.Errorf("mcp: %s directory: %w", what, err)
		}
	}
	switch fi, err := os.Lstat(path); {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return fmt.Errorf("mcp: %s file: %w", what, err)
	case fi.Mode()&fs.ModeSymlink != 0:
		return fmt.Errorf("mcp: %s file %s is a symlink; refusing to write it through it", what, path)
	case !fi.Mode().IsRegular():
		return fmt.Errorf("mcp: %s file %s is not a regular file", what, path)
	}

	f, err := os.CreateTemp(dir, ".mcp-"+what+"-*")
	if err != nil {
		return fmt.Errorf("mcp: %s file: %w", what, err)
	}
	tmp := f.Name()
	done := false
	defer func() {
		if !done {
			_ = f.Close()
			_ = os.Remove(tmp)
		}
	}()
	if err := f.Chmod(0o600); err != nil && runtime.GOOS != goosWindows {
		return fmt.Errorf("mcp: %s file: %w", what, err)
	}
	if _, err := f.WriteString(line + "\n"); err != nil {
		return fmt.Errorf("mcp: %s file: %w", what, err)
	}
	if err := f.Sync(); err != nil {
		return fmt.Errorf("mcp: %s file: %w", what, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("mcp: %s file: %w", what, err)
	}
	// rename replaces a symlink that appeared since the check instead of following it.
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("mcp: %s file: %w", what, err)
	}
	done = true
	return nil
}

// loadToken returns the token kept in the token file at path, so that a client registered once
// keeps working across Cogpit's starts (F6). It returns "" with the reason in dropped when the file
// holds nothing Cogpit can keep using: no token Cogpit wrote, or a file other users may read (a
// token they may have read is not kept); a missing file is "" without a reason. A symlink, or
// anything but a regular file, is an error, as writeTokenFile refuses to write through one.
func loadToken(path string) (token, dropped string, err error) {
	fi, err := os.Lstat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return "", "", nil
	case err != nil:
		return "", "", fmt.Errorf("mcp: token file: %w", err)
	case fi.Mode()&fs.ModeSymlink != 0:
		return "", "", fmt.Errorf("mcp: token file %s is a symlink; refusing to read a token through it", path)
	case !fi.Mode().IsRegular():
		return "", "", fmt.Errorf("mcp: token file %s is not a regular file", path)
	case runtime.GOOS != goosWindows && fi.Mode().Perm()&0o077 != 0:
		return "", fmt.Sprintf("the token file was accessible by other users (mode %#o)", fi.Mode().Perm()), nil
	}
	line, err := readLine(path)
	if err != nil {
		return "", "", err
	}
	if !validToken(line) {
		return "", "the token file did not hold a token Cogpit wrote", nil
	}
	return line, "", nil
}

// validToken reports whether s is a token newToken makes: 32 hex-encoded bytes.
func validToken(s string) bool {
	b, err := hex.DecodeString(s)
	return err == nil && len(b) == tokenBytes
}

// removeIfHolds removes path if it is a regular file that still holds line: another Cogpit
// started later may have replaced it with its own, which must survive this one's exit, and a
// symlink is not Cogpit's either. what names the file in errors.
func removeIfHolds(path, line, what string) error {
	if path == "" || line == "" {
		return nil
	}
	fi, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("mcp: remove %s file: %w", what, err)
	}
	if !fi.Mode().IsRegular() {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("mcp: remove %s file: %w", what, err)
	}
	if subtle.ConstantTimeCompare(bytes.TrimSpace(data), []byte(line)) != 1 {
		return nil
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("mcp: remove %s file: %w", what, err)
	}
	return nil
}
