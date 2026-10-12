package kubeconfig

import (
	"errors"
	"fmt"
	"strings"
)

// MaxFileSize bounds a kubeconfig Cogpit reads from a source or accepts pasted. Real kubeconfigs
// with many contexts and embedded certificates stay far below it.
const MaxFileSize = 1 << 20

// ErrNoContexts is returned for a kubeconfig document that defines no context: there is nothing
// to connect to, and a sync folder's other files (notes, backups) look like that.
var ErrNoContexts = errors.New("kubeconfig: the document defines no context")

// Pasted checks kubeconfig text the user pasted: at most MaxFileSize bytes, one valid kubeconfig
// document, at least one context. It returns the contexts. Its errors never echo the text.
func Pasted(data []byte) ([]Context, error) {
	if len(data) > MaxFileSize {
		return nil, fmt.Errorf("kubeconfig: %d bytes is more than a kubeconfig (at most %d)", len(data), MaxFileSize)
	}
	if strings.TrimSpace(string(data)) == "" {
		return nil, errors.New("kubeconfig: the text is empty")
	}
	ctxs, err := Contexts(data)
	if err != nil {
		return nil, err
	}
	if len(ctxs) == 0 {
		return nil, ErrNoContexts
	}
	return ctxs, nil
}

// SyncCandidate reports whether a file of this name in a sync folder is read as a kubeconfig:
// hidden files, editor backups and swap files are not. Files that pass are still read only if
// they parse and define a context.
func SyncCandidate(name string) bool {
	switch {
	case name == "", strings.HasPrefix(name, "."), strings.HasSuffix(name, "~"),
		strings.HasSuffix(name, ".swp"), strings.HasSuffix(name, ".tmp"):
		return false
	}
	return true
}
