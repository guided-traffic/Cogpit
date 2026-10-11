//go:build !darwin

package fyneui

// titlebar does nothing outside macOS: Linux and Windows keep their native window frames
// (titlebar.go).
type titlebar struct{}

func newTitlebar(*shell) *titlebar { return &titlebar{} }

func (*titlebar) install()   {}
func (*titlebar) mouseDown() {}
func (*titlebar) stop()      {}
