//go:build darwin

package fyneui

/*
#cgo LDFLAGS: -framework AppKit -framework Foundation
#include <stdlib.h>
#include <string.h>
#include "clipboard_darwin.h"
*/
import "C"

import "unsafe"

// setConcealedClipboard puts a Secret's value on the clipboard marked concealed and transient
// (nspasteboard.org's types: clipboard managers neither show nor record it) and for this Mac
// only (not handed to other devices through Universal Clipboard). It reports false when it could
// not, and the caller falls back to Fyne's clipboard. Main goroutine only.
func setConcealedClipboard(text string) bool {
	cs := C.CString(text)
	defer func() { // the C copy holds the value: zero it before it is freed
		C.memset(unsafe.Pointer(cs), 0, C.size_t(len(text)))
		C.free(unsafe.Pointer(cs))
	}()
	return bool(C.cogpitSetConcealedString(cs))
}
