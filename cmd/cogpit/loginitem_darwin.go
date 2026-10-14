//go:build darwin

package main

/*
#cgo LDFLAGS: -framework Foundation -framework ServiceManagement
#include <stdlib.h>
#include "loginitem_darwin.h"
*/
import "C"

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// errNotInAppBundle refuses start at login to a Cogpit that does not run from its app bundle (go
// run, bin/cogpit): SMAppService.mainAppService describes an app bundle, so Cogpit does not ask it
// about a bare executable at all.
var errNotInAppBundle = errors.New("this Cogpit does not run from its app bundle (a development build started as " +
	"bin/cogpit or with go run): start at login needs Cogpit.app, signed and in Applications")

// macLoginItem is Cogpit.app's own login item (SMAppService.mainAppService; D-48).
type macLoginItem struct {
	inBundle bool
}

// newLoginItem returns macOS's login item of this app for the engine (Options.LoginItem).
func newLoginItem() engine.LoginItem {
	return macLoginItem{inBundle: C.cogpitLoginItemInAppBundle() != 0}
}

// Status reads SMAppService's status; outside an app bundle, not found without asking.
func (l macLoginItem) Status() string {
	if !l.inBundle {
		return engine.LoginItemNotFound
	}
	return loginItemStatus(int(C.cogpitLoginItemStatus()))
}

// loginItemStatus names SMAppServiceStatus's values.
func loginItemStatus(raw int) string {
	switch raw {
	case 0: // SMAppServiceStatusNotRegistered
		return engine.LoginItemOff
	case 1: // SMAppServiceStatusEnabled
		return engine.LoginItemOn
	case 2: // SMAppServiceStatusRequiresApproval
		return engine.LoginItemNeedsApproval
	}
	return engine.LoginItemNotFound // SMAppServiceStatusNotFound, and anything newer
}

// Register registers the app as a login item.
func (l macLoginItem) Register() error {
	if !l.inBundle {
		return errNotInAppBundle
	}
	return smError(C.cogpitLoginItemRegister())
}

// Unregister removes the app's login item.
func (l macLoginItem) Unregister() error {
	if !l.inBundle {
		return nil // nothing was ever registered for a bare executable
	}
	return smError(C.cogpitLoginItemUnregister())
}

// OpenSettings opens System Settings › General › Login Items.
func (macLoginItem) OpenSettings() error {
	C.cogpitLoginItemOpenSettings()
	return nil
}

// maxSMError bounds the text of an error macOS gives.
const maxSMError = 300

// smError turns ServiceManagement's error text (or nil) into an error, one line and bounded, and
// frees it.
func smError(cs *C.char) error {
	if cs == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(cs))
	return errors.New(oneLine(C.GoString(cs), maxSMError))
}

// oneLine makes text one line of at most n bytes, without control characters.
func oneLine(text string, n int) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return ' '
		}
		return r
	}, text)
	if len(text) > n {
		cut := n
		for cut > 0 && !utf8.RuneStart(text[cut]) {
			cut--
		}
		text = text[:cut] + "…"
	}
	return strings.TrimSpace(text)
}
