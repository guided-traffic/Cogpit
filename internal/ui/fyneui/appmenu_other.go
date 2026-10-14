//go:build !darwin

package fyneui

// installAbout does nothing outside macOS: Windows and Linux have no application menu, and Help ends
// with About Cogpit there (menus.go).
func (*shell) installAbout() {}
