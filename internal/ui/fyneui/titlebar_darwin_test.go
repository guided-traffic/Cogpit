//go:build darwin

package fyneui

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// terminatedWindow is a window after GLFW terminated: asking for its native handle panics, as
// GLFW's GetCocoaWindow does then.
type terminatedWindow struct{ fyne.Window }

func (terminatedWindow) RunNative(func(any)) {
	panic("NotInitialized: The GLFW library is not initialized")
}

// The shell stops the title bar after the app's run ended; that must not ask GLFW for the window
// (every quit panicked there, 2026-10-12).
func TestTitlebarStopAfterTheRunEnded(t *testing.T) {
	test.NewTempApp(t)
	tb := newTitlebar(&shell{win: terminatedWindow{test.NewTempWindow(t, nil)}})
	tb.window = 1
	titlebars.Store(tb.id, tb)

	tb.stop()

	if _, ok := titlebars.Load(tb.id); ok || tb.window != 0 {
		t.Errorf("still registered %v, window %#x", ok, tb.window)
	}
}
