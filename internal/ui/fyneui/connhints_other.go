//go:build !darwin

package fyneui

// watchConnectionHints has no signals off macOS: the heartbeat and the broken streams find an
// interrupted connection there (D-58).
func watchConnectionHints(func(connHint)) {}
