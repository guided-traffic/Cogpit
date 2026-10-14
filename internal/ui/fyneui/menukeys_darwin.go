//go:build darwin

package fyneui

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <stdbool.h>
#include "menukeys_darwin.h"
*/
import "C"

// platformMenuKey reports whether macOS's main menu runs an item for its key equivalent rather
// than for a click (menukeys.go): AppKit's current event is a key's. Main thread only; false
// elsewhere.
func platformMenuKey() bool { return bool(C.cogpitMenuKeyEvent()) }
