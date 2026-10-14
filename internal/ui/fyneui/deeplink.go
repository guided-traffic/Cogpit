package fyneui

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// cogpit:// links (D-47; engine/deeplink.go). macOS hands the app every link opened anywhere; the
// desktop binary queues them (engine.DeepLinkQueue) until the window is up, and the window then
// hands each to the engine, which navigates through its ui.* commands with origin deeplink, says
// in a toast what it opened or why it did not, and audits it. The window's part is to bring
// itself to the front, so that the user sees what a link did, or that it was refused.

// deepLinkOpener is the part of the engine that opens a link (engine.Session.OpenDeepLink); a
// test's fake engine without it leaves links alone.
type deepLinkOpener interface {
	OpenDeepLink(ctx context.Context, raw string) (engine.DeepLinkResult, error)
}

// attachDeepLinks opens the links of q, those that came before the window and every later one, one
// at a time on the queue's goroutine (a link that connects a cluster takes seconds), each first
// bringing the window to the front. It returns the function that detaches the window.
func (sh *shell) attachDeepLinks(q *engine.DeepLinkQueue) (detach func()) {
	op, ok := sh.eng.(deepLinkOpener)
	if q == nil || !ok {
		return func() {}
	}
	return q.Attach(func(raw string) {
		if sh.ctx.Err() != nil {
			return
		}
		sh.do(sh.raiseForLink)
		_, _ = op.OpenDeepLink(sh.ctx, raw) // the engine's toast says what it opened, or why not
	})
}

// raiseForLink brings the window to the front for a link (Fyne's RequestFocus: on macOS the app is
// activated and the window made key). Main goroutine.
func (sh *shell) raiseForLink() {
	sh.win.Show()
	sh.win.RequestFocus()
}
