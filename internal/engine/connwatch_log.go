package engine

import (
	"fmt"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// The connection watch's lines in the connection log (D-66): an interruption begins a reconnect
// attempt, with the error picture of the check that failed; its tries are written when their
// reason changes (a reconnect that waits all night writes a line per change, not one every 30
// seconds); it ends when the connection is back or the login expired. A login the API server
// rejects begins a login attempt (connwatch.go), and a credential plugin that failed a check adds
// its scrubbed standard error to the attempt that runs.

// logInterrupted begins the reconnect attempt of an interruption with its cause.
func (w *connWatch) logInterrupted(reason string, cause error) {
	ctx := w.b.Context
	logs := w.logs()
	a := logs.begin(ctx, AttemptReconnect, LogWarn, "Connection interrupted: "+reason+"; Cogpit is reconnecting")
	for _, line := range cluster.ErrorPicture(cause) {
		logs.stepTo(a, LogWarn, cluster.StepError, line, 0)
	}
	if text := pluginStderrLines(cause); text != "" {
		logs.stepTo(a, LogWarn, cluster.StepPlugin, text, 0)
	}
	w.mu.Lock()
	w.loggedReason = reason
	w.mu.Unlock()
}

// logTry writes a failed reconnect try when its reason differs from the last one written.
func (w *connWatch) logTry(reason string, next time.Duration) {
	w.mu.Lock()
	same := reason == w.loggedReason
	w.loggedReason = reason
	w.mu.Unlock()
	if same {
		return
	}
	w.logs().add(w.b.Context, LogWarn, fmt.Sprintf("A try to reach the cluster failed: %s; the next in %s (tries "+
		"with the same reason are not written again)", reason, next))
}

// logPluginStderr adds a failed credential plugin's scrubbed standard error to the attempt that
// runs (a reconnect, a login). A check of a connection that stands has none: the interruption it
// may lead to writes its cause's (logInterrupted).
func (w *connWatch) logPluginStderr(err error) {
	if text := pluginStderrLines(err); text != "" {
		w.logs().stepRunning(w.b.Context, LogWarn, cluster.StepPlugin, text)
	}
}
