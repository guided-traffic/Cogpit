//go:build darwin

package fyneui

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <stdlib.h>
#include "appmenu_darwin.h"
*/
import "C"

import (
	"log/slog"
	"sync/atomic"
	"unsafe"
)

// macOS's application menu has About Cogpit (A60), showing Cogpit's own dialog (about.go): GLFW
// creates the menu with "About <app name>" opening AppKit's standard panel, and installAbout points
// that item at the shell instead, under Cogpit's title. Fyne's own way, a menu item labelled
// "About", takes the same item over the first time and loses it on the next rebuild of the menu bar,
// which Cogpit does whenever a cluster connects or disconnects (shell.go, SetMainMenu).

// aboutShell is the shell whose About the application menu shows: the app has one window.
var aboutShell atomic.Pointer[shell]

// installAbout puts Cogpit's About into the application menu. Call it on the main goroutine once the
// app started (GLFW has made the menu bar by then); later calls only find the item again.
func (sh *shell) installAbout() {
	aboutShell.Store(sh)
	title := C.CString(aboutItemLabel)
	defer C.free(unsafe.Pointer(title))
	if !bool(C.cogpitAppMenuInstallAbout(title)) {
		slog.Warn("macOS has no application menu yet: no About Cogpit in it")
	}
}

// cogpitAboutChosen is AppKit's report that the user chose About Cogpit. It runs on the main thread,
// inside AppKit's dispatch of the menu event; the dialog opens on Fyne's main goroutine.
//
//export cogpitAboutChosen
func cogpitAboutChosen() {
	if sh := aboutShell.Load(); sh != nil {
		sh.do(sh.showAbout)
	}
}
