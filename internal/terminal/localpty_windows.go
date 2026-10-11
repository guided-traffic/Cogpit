//go:build windows

package terminal

import "fmt"

// startLocal is not implemented on Windows yet: it needs a ConPTY session (the approach
// fyne-io/terminal's term_windows.go takes) with the same per-terminal kubeconfig.
func startLocal(opts LocalOptions) (Session, error) {
	_ = opts
	return nil, fmt.Errorf("local terminal on Windows (ConPTY): %w", ErrNotImplemented)
}
