//go:build darwin

package fyneui

/*
#cgo LDFLAGS: -framework AppKit
#include "reducemotion_darwin.h"
*/
import "C"

// reduceMotion reports macOS's Reduce motion (System Settings › Accessibility › Display): the
// connection's dots then do not blink (conndot.go). A property read, cheap: the blinker reads it
// whenever a blink starts, so a change shows with the next (re)connect.
func reduceMotion() bool { return C.cogpitReduceMotion() != 0 }
