//go:build darwin && cgo

package update

/*
#cgo CFLAGS: -fno-objc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#include "sparkle_darwin.h"
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

// detailSize bounds the text the Objective-C side hands back.
const detailSize = 512

// call runs f with a buffer for the detail text and returns its result and the text.
func call(f func(buf *C.char, size C.size_t) C.int) (int, string) {
	buf := (*C.char)(C.calloc(detailSize, 1))
	defer C.free(unsafe.Pointer(buf))
	code := f(buf, detailSize)
	return int(code), C.GoString(buf)
}

// load checks the bundle and loads Sparkle.framework (sparkle_darwin.m).
func load() error {
	code, detail := call(func(buf *C.char, size C.size_t) C.int { return C.cogpitSparkleLoad(buf, size) })
	switch code {
	case C.COGPIT_SPARKLE_OK:
		return nil
	case C.COGPIT_SPARKLE_NO_BUNDLE:
		return ErrNoBundle
	case C.COGPIT_SPARKLE_NOT_INSTALLED:
		return ErrNotInstalled
	case C.COGPIT_SPARKLE_NO_FRAMEWORK:
		return ErrNoFramework
	case C.COGPIT_SPARKLE_NO_FEED:
		return ErrNoFeed
	case C.COGPIT_SPARKLE_NO_KEY:
		return ErrNoKey
	}
	return fmt.Errorf("the updater could not be loaded: %s", detail)
}

// startFailure is why Sparkle did not start, once it tried.
func startFailure() error {
	code, detail := call(func(buf *C.char, size C.size_t) C.int { return C.cogpitSparkleState(buf, size) })
	if code == C.COGPIT_SPARKLE_FAILED {
		return errors.New("the updater did not start: " + detail)
	}
	return nil
}

func start(automaticChecks bool) { C.cogpitSparkleStart(C.bool(automaticChecks), C.bool(true)) }

func check() { C.cogpitSparkleCheck() }

func setAutomaticChecks(on bool) { C.cogpitSparkleSetAutomaticChecks(C.bool(on)) }
