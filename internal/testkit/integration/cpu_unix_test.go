//go:build darwin || linux

package integration_test

import (
	"syscall"
	"time"
)

// processCPU returns the CPU time (user and system) the process used so far.
func processCPU() (time.Duration, bool) {
	var ru syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &ru); err != nil {
		return 0, false
	}
	return time.Duration(ru.Utime.Nano() + ru.Stime.Nano()), true
}
