//go:build !darwin

package fyneui

// platformMenuKey is false outside macOS: Fyne's own menu never runs a window key's item for a key
// (its shownShortcut matches no typed key), so every run is a click (menukeys.go).
func platformMenuKey() bool { return false }
