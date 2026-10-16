//go:build !darwin && !linux && !windows

package cluster

import "errors"

// childProcesses finds no process where Cogpit cannot list them: a credential plugin's run that
// Cogpit ends there runs on, and its login is not used (loginrun.go).
func childProcesses() []childProcess { return nil }

// signalPlugin cannot end a process here.
func signalPlugin(int, bool) error { return errors.New("not supported on this system") }
