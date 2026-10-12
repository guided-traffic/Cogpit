//go:build !(darwin || linux)

package integration_test

import "time"

// processCPU is not measured on this platform.
func processCPU() (time.Duration, bool) { return 0, false }
