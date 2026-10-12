//go:build !darwin

package fyneui

// setConcealedClipboard is macOS only so far: elsewhere a Secret's value goes on the clipboard
// through Fyne, unmarked (H-25).
func setConcealedClipboard(string) bool { return false }
