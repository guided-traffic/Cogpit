//go:build darwin || linux

package cluster

import "syscall"

// signalPlugin ends a credential plugin's process: SIGTERM, or SIGKILL when kill is set.
func signalPlugin(pid int, kill bool) error {
	sig := syscall.SIGTERM
	if kill {
		sig = syscall.SIGKILL
	}
	return syscall.Kill(pid, sig)
}
