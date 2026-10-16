//go:build darwin && cgo

package nativedialog

/*
#cgo CFLAGS: -fno-objc-arc
#cgo LDFLAGS: -framework AppKit -framework Foundation -framework UniformTypeIdentifiers
#include <stdlib.h>
#include "panel_darwin.h"
*/
import "C"

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"
)

// errSize bounds the reason texts the Objective-C side hands back.
const errSize = 512

// request is one panel on the screen: what it picked so far and whom to tell.
type request struct {
	picks []Pick
	done  func([]Pick)
}

var (
	requestsMu sync.Mutex
	requests   = map[uint64]*request{} // by the id the panel's callbacks name
	requestIDs atomic.Uint64
)

// Open shows the system's open panel as a sheet on window, an NSWindow pointer
// (driver.MacWindowContext), and returns at once. When the user ends it, done runs on the main
// thread, inside AppKit's event handling, with the chosen items, or with none for a cancel; hand
// widget work on to fyne.Do. Call it on the main thread. It returns ErrNoWindow, ErrBusy or
// ErrNotMain when no panel was shown, and done is not called.
func Open(window uintptr, o OpenOptions, done func([]Pick)) error {
	opts, free := cOptions(o.Title, o.Message, o.Prompt, o.Dir, "", o.Types)
	defer free()
	opts.files, opts.folders = cBool(o.Files), cBool(o.Folders)
	opts.multiple, opts.hidden = cBool(o.Multiple), cBool(o.ShowHidden)
	opts.bookmark = C.int(o.Bookmark)
	return show(window, done, func(id C.uintptr_t) C.int {
		return C.cogpitPanelOpen(C.uintptr_t(window), opts, id)
	})
}

// Save shows the system's save panel as a sheet on window and returns at once; done gets the
// chosen path on the main thread, never for a cancel. The panel asks before an existing file is
// replaced; the caller writes the file. Errors as for Open.
func Save(window uintptr, o SaveOptions, done func(path string)) error {
	opts, free := cOptions(o.Title, o.Message, o.Prompt, o.Dir, o.Name, o.Types)
	defer free()
	return show(window, func(picks []Pick) {
		if len(picks) > 0 {
			done(picks[0].Path)
		}
	}, func(id C.uintptr_t) C.int {
		return C.cogpitPanelSave(C.uintptr_t(window), opts, id)
	})
}

// show registers done under a new id and runs begin, which shows the panel.
func show(window uintptr, done func([]Pick), begin func(C.uintptr_t) C.int) error {
	if window == 0 {
		return ErrNoWindow
	}
	id := requestIDs.Add(1)
	requestsMu.Lock()
	requests[id] = &request{done: done}
	requestsMu.Unlock()
	code := begin(C.uintptr_t(id))
	if code == C.COGPIT_PANEL_SHOWN {
		return nil
	}
	requestsMu.Lock()
	delete(requests, id)
	requestsMu.Unlock()
	switch code {
	case C.COGPIT_PANEL_BUSY:
		return ErrBusy
	case C.COGPIT_PANEL_NOT_MAIN:
		return ErrNotMain
	default:
		return ErrNoWindow
	}
}

// cOptions builds the C options; free releases their strings once the call returned (the
// Objective-C side copies them).
func cOptions(title, message, prompt, dir, name string, types []string) (*C.cogpitPanelOptions, func()) {
	o := (*C.cogpitPanelOptions)(C.calloc(1, C.size_t(unsafe.Sizeof(C.cogpitPanelOptions{}))))
	var strs []*C.char
	cs := func(s string) *C.char {
		if s == "" {
			return nil
		}
		p := C.CString(s)
		strs = append(strs, p)
		return p
	}
	o.title, o.message, o.prompt = cs(title), cs(message), cs(prompt)
	o.directory, o.name = cs(dir), cs(name)
	o.types = cs(strings.Join(types, "\n"))
	return o, func() {
		for _, p := range strs {
			C.free(unsafe.Pointer(p))
		}
		C.free(unsafe.Pointer(o))
	}
}

func cBool(b bool) C.int {
	if b {
		return 1
	}
	return 0
}

// cogpitPanelPicked is a panel's report of one chosen item, on the main thread.
//
//export cogpitPanelPicked
func cogpitPanelPicked(id C.uintptr_t, path *C.char, folder C.int, bookmark unsafe.Pointer, length C.size_t, reason *C.char) {
	requestsMu.Lock()
	defer requestsMu.Unlock()
	r := requests[uint64(id)]
	if r == nil {
		return
	}
	p := Pick{Path: C.GoString(path), Folder: folder != 0}
	if bookmark != nil && length > 0 {
		p.Bookmark = C.GoBytes(bookmark, C.int(length))
	}
	if reason != nil {
		p.BookmarkErr = C.GoString(reason)
	}
	r.picks = append(r.picks, p)
}

// cogpitPanelDone is a panel's end, on the main thread: done gets what was picked.
//
//export cogpitPanelDone
func cogpitPanelDone(id C.uintptr_t, ok C.int) {
	requestsMu.Lock()
	r := requests[uint64(id)]
	delete(requests, uint64(id))
	requestsMu.Unlock()
	if r == nil {
		return
	}
	if ok == 0 {
		r.picks = nil
	}
	r.done(r.picks)
}

// makeBookmark makes a security-scoped bookmark of path: what the panel makes of each pick, for
// tests.
func makeBookmark(path string, access Access) ([]byte, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	reason := (*C.char)(C.calloc(errSize, 1))
	defer C.free(unsafe.Pointer(reason))
	var data unsafe.Pointer
	var length C.size_t
	if C.cogpitBookmarkCreate(cpath, C.int(access), &data, &length, reason, errSize) == 0 {
		return nil, errors.New(C.GoString(reason))
	}
	defer C.cogpitFree(data)
	return C.GoBytes(data, C.int(length)), nil
}

// resolveBookmark resolves a security-scoped bookmark and starts accessing its item for the
// process's life (Store.Restore).
func resolveBookmark(data []byte, access Access) (resolved, error) {
	if len(data) == 0 {
		return resolved{}, errors.New("an empty bookmark")
	}
	cdata := C.CBytes(data)
	defer C.free(cdata)
	reason := (*C.char)(C.calloc(errSize, 1))
	defer C.free(unsafe.Pointer(reason))
	var (
		path      *C.char
		accessing C.int
		fresh     unsafe.Pointer
		freshLen  C.size_t
	)
	if C.cogpitBookmarkResolve(cdata, C.size_t(len(data)), C.int(access), &path, &accessing, &fresh, &freshLen, reason, errSize) == 0 {
		return resolved{}, errors.New(C.GoString(reason))
	}
	r := resolved{accessing: accessing != 0}
	if path != nil {
		r.path = C.GoString(path)
		C.cogpitFree(unsafe.Pointer(path))
	}
	if fresh != nil {
		r.fresh = C.GoBytes(fresh, C.int(freshLen))
		C.cogpitFree(fresh)
	}
	return r, nil
}
