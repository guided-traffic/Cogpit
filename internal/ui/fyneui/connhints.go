package fyneui

// connHint is a signal of the operating system that a connection may have broken (D-58).
type connHint int

const (
	hintWake    connHint = 1 // the computer woke from sleep
	hintNetwork connHint = 2 // the network path changed
)

// startConnectionHints hands the operating system's wake and network signals to the engine, which
// checks the connection at once (S2) and tries an interrupted one at once (S5). The window's only;
// a shell over a fake engine has none.
func (sh *shell) startConnectionHints() {
	if sh.sess == nil {
		return
	}
	s := sh.sess
	watchConnectionHints(func(h connHint) {
		switch h {
		case hintWake:
			s.NoteWake()
		case hintNetwork:
			s.NoteNetworkChange()
		}
	})
}
