//go:build !darwin

package fyneui

// reduceMotion is false outside macOS: Cogpit reads no such setting on Linux and Windows yet
// (conndot.go).
func reduceMotion() bool { return false }
