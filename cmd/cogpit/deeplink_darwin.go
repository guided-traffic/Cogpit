//go:build darwin

package main

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework CoreServices
#include "deeplink_darwin.h"
*/
import "C"

import (
	"log/slog"
	"sync/atomic"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// maxReceivedLink is COGPIT_MAX_DEEPLINK: the most of a link the Apple event handler hands over.
const maxReceivedLink = C.COGPIT_MAX_DEEPLINK

// deepLinks is the queue the Apple event handler fills (cogpitDeepLinkReceived).
var deepLinks atomic.Pointer[engine.DeepLinkQueue]

// installDeepLinks installs the handler of macOS's "open URL" Apple event, which delivers the
// cogpit:// links the app bundle's CFBundleURLTypes registers (D-47), and returns the queue they
// wait in until the window opens them. Call it on the main thread before the app runs: main does,
// before fyneui.Run, and Fyne keeps main on the main thread. Nothing is registered with Launch
// Services here: the scheme is the app bundle's Info.plist's (Makefile, app).
func installDeepLinks(log *slog.Logger) *engine.DeepLinkQueue {
	q := engine.NewDeepLinkQueue(log)
	deepLinks.Store(q)
	C.cogpitDeepLinkInstall()
	return q
}

// cogpitDeepLinkReceived is the Apple event handler's report of a link, at most
// COGPIT_MAX_DEEPLINK bytes of it. It runs on the main thread inside AppKit's event dispatch, so it
// only queues the link: it never blocks and never opens it here.
//
//export cogpitDeepLinkReceived
func cogpitDeepLinkReceived(url *C.char, length C.int) {
	receivedLink(deepLinks.Load(), url, length)
}

// receivedLink queues length bytes of url, at most maxReceivedLink.
func receivedLink(q *engine.DeepLinkQueue, url *C.char, length C.int) {
	if q == nil || url == nil || length < 0 || length > maxReceivedLink {
		return
	}
	q.Deliver(C.GoStringN(url, length))
}
