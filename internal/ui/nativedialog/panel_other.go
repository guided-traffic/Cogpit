//go:build !darwin || !cgo

package nativedialog

// Open has no system panel in this build: the caller shows its own dialog.
func Open(uintptr, OpenOptions, func([]Pick)) error { return ErrUnsupported }

// Save has no system panel in this build: the caller shows its own dialog.
func Save(uintptr, SaveOptions, func(string)) error { return ErrUnsupported }

// resolveBookmark: no bookmarks in this build; the store keeps what it holds (none is ever made
// here).
func resolveBookmark([]byte, Access) (resolved, error) { return resolved{}, ErrUnsupported }
