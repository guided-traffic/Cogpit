//go:build darwin

package fyneui

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework Network
#include "connhints_darwin.h"
*/
import "C"

import "sync"

// macOS's wake and network signals for the connection watch (D-58, S2 and S5): the computer woke
// (NSWorkspaceDidWakeNotification) and its network path changed (Network.framework's
// nw_path_monitor: Wi-Fi, Ethernet, a VPN). Both are cheap: one observer and one monitor for the
// process, their events handed to the engine, which checks the connection at once.

var (
	hintMu   sync.Mutex
	hintFn   func(connHint)
	hintOnce sync.Once
)

// cogpitConnectionHint is called by the Objective-C side on AppKit's main thread (wake) or a
// dispatch queue (network). It must not block.
//
//export cogpitConnectionHint
func cogpitConnectionHint(kind C.int) {
	hintMu.Lock()
	fn := hintFn
	hintMu.Unlock()
	if fn != nil {
		fn(connHint(kind))
	}
}

// watchConnectionHints hands macOS's wake and network changes to fn from now on; the observers start
// once per process.
func watchConnectionHints(fn func(connHint)) {
	hintMu.Lock()
	hintFn = fn
	hintMu.Unlock()
	hintOnce.Do(func() { C.cogpitWatchConnectionHints() })
}
