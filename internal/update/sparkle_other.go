//go:build !darwin || !cgo

package update

// Elsewhere there is no updater yet (Windows and Linux with M11).

func load() error             { return ErrNotMacOS }
func startFailure() error     { return nil }
func start(bool)              {}
func check()                  {}
func setAutomaticChecks(bool) {}
