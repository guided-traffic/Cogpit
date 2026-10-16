package fyneui

import (
	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
	"github.com/guided-traffic/cogpit/internal/model"
)

// loginWaitLine says what a connection waits for while its credential plugin runs (D-73; the
// engine's Connection.WaitingForLogin, as agents read it in ui_state): the program, since when, and
// when Cogpit ends it. "" while none runs. The connecting workspace's placeholder (tableview.go) and
// the dots' tooltips (conndot.go) show it.
func loginWaitLine(w engine.LoginWait, tf model.TimeFormat) string {
	if w.Program == "" {
		return ""
	}
	return "Waiting for the login through " + kubeconfig.DisplayText(w.Program, maxShownArg) + " since " + tf.FormatClock(w.Since) +
		"; it may wait for you in your browser. Cogpit ends it at " + tf.FormatClock(w.Until) + "."
}
