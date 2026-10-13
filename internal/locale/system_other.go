//go:build !darwin || !cgo

package locale

// systemFormat has no system lookup here: Linux, and macOS without cgo (the pure-Go cogpit-cli),
// read the environment's locale (Environment). Windows does too for now: its region settings
// (GetUserDefaultLocaleName, GetLocaleInfoEx) are not read, and without LANG it gets ISO 8601.
func systemFormat() (Format, bool) { return Format{}, false }
