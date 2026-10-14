//go:build !darwin

package main

// readOnlyVolume counts every path as read-only: only the macOS app bundle carries cogpit-cli
// (D-16), and bundledCLI looks for it on macOS only; Windows and Linux packages come with M11.
func readOnlyVolume(string) bool { return true }
