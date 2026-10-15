package fyneui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// connBanner is the thin strip over the view while the connected cluster's connection is
// interrupted or its login expired (D-58, S3): what is interrupted, since when, Cogpit's next
// attempt, and Reconnect now (cluster.reconnect); for an expired login Log in again (cluster.login).
// It reads only the UI state, so it says what an agent reads in ui_state. The window stays usable:
// the views keep the cluster's state as of the interruption, and writes are disabled with the
// reason in their menus and buttons. Lens's behaviour on a lost connection is not captured: this is
// Cogpit's own.
type connBanner struct {
	sh     *shell
	root   *fyne.Container // hidden while the connection works
	title  *textLine
	detail *textLine
	action *widget.Button
	icon   *widget.Icon
	tint   *bannerTint
	ticker *time.Ticker // the countdown to the next attempt; nil while hidden
	stop   chan struct{}
}

func newConnBanner(sh *shell) *connBanner {
	b := &connBanner{sh: sh}
	b.title = newTextLine("", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	b.title.truncate = true
	b.detail = newTextLine("", colorMuted, theme.SizeNameText, fyne.TextStyle{})
	b.detail.truncate = true
	b.action = widget.NewButtonWithIcon("Reconnect now", theme.ViewRefreshIcon(), b.act)
	b.action.Importance = widget.HighImportance
	b.icon = widget.NewIcon(themedIcon(theme.WarningIcon(), colorStatusWarning))
	b.tint = newBannerTint(colorStatusWarning)
	content := container.NewBorder(nil, nil, container.NewCenter(b.icon), container.NewCenter(b.action),
		container.NewVBox(b.title, b.detail))
	b.root = container.NewVBox(container.NewStack(b.tint, inset(theme.SizeNameInnerPadding, theme.SizeNamePadding, content)), newDivider(false))
	b.root.Hide()
	return b
}

// update follows the UI state's connection. Main goroutine only.
func (b *connBanner) update() {
	c := b.sh.ui.Connection
	switch c.State {
	case engine.Interrupted, engine.LoginExpired:
	default:
		b.hide()
		return
	}
	b.fill(c)
	if !b.root.Visible() {
		b.root.Show()
		b.startTicker()
	}
	b.root.Refresh()
}

// fill writes the banner's lines and its button for c.
func (b *connBanner) fill(c engine.Connection) {
	name, _ := splitContextID(b.sh.ui.Context)
	tf := b.sh.timeFormat()
	since := tf.FormatClock(c.Since)
	if c.State == engine.LoginExpired {
		b.title.SetText(fmt.Sprintf("The login to %s expired · since %s", name, since))
		b.detail.SetText(loginDetail(c, since))
		b.tint.color = colorStatusError
		b.icon.SetResource(themedIcon(theme.ErrorIcon(), colorStatusError))
		b.action.SetText("Log in again")
		b.action.SetIcon(theme.LoginIcon())
		if c.LoggingIn {
			b.action.SetText("Logging in…")
			b.action.Disable()
		} else {
			b.action.Enable()
		}
		b.tint.Refresh()
		return
	}
	b.title.SetText(fmt.Sprintf("The connection to %s is interrupted · since %s", name, since))
	b.detail.SetText(interruptedDetail(c, since, b.sh.now()))
	b.tint.color = colorStatusWarning
	b.icon.SetResource(themedIcon(theme.WarningIcon(), colorStatusWarning))
	b.action.SetText("Reconnect now")
	b.action.SetIcon(theme.ViewRefreshIcon())
	if c.NextAttempt.IsZero() {
		b.action.Disable() // an attempt runs now
	} else {
		b.action.Enable()
	}
	b.tint.Refresh()
}

// interruptedDetail is the second line while interrupted: why, the next attempt, what the window
// shows meanwhile.
func interruptedDetail(c engine.Connection, since string, now time.Time) string {
	next := "trying now…"
	if !c.NextAttempt.IsZero() {
		next = "next attempt " + attemptIn(c.NextAttempt.Sub(now))
	}
	parts := []string{"Cogpit reconnects by itself: " + next,
		"the views show the cluster as of " + since, "changes are disabled until it is back"}
	if r := strings.TrimSpace(c.Reason); r != "" {
		parts = append([]string{upperFirst(r)}, parts...)
	}
	return strings.Join(parts, " · ")
}

// loginDetail is the second line while the login is expired.
func loginDetail(c engine.Connection, since string) string {
	what := "Log in again runs the login program once; it may open your browser"
	if c.NeedsApproval {
		what = "The login program changed since you allowed it: Log in again connects anew and shows it to you first"
	}
	parts := []string{what, "the views show the cluster as of " + since, "changes are disabled"}
	if r := strings.TrimSpace(c.Reason); r != "" {
		parts = append([]string{upperFirst(r)}, parts...)
	}
	return strings.Join(parts, " · ")
}

// attemptIn says how long until the next attempt: "in 8 s", "in 1 min", "in a moment".
func attemptIn(d time.Duration) string {
	switch s := int(d.Round(time.Second) / time.Second); {
	case s <= 0:
		return "in a moment"
	case s < 60:
		return fmt.Sprintf("in %d s", s)
	default:
		return fmt.Sprintf("in %d min", (s+30)/60)
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// act is the button: Reconnect now, or Log in again.
func (b *connBanner) act() {
	switch b.sh.ui.Connection.State {
	case engine.Interrupted:
		b.sh.dispatch("cluster.reconnect", struct{}{}, nil)
	case engine.LoginExpired:
		name := b.sh.ui.Context
		b.sh.dispatch("cluster.login", struct{}{}, func(_ any, err error) {
			switch {
			case errors.Is(err, engine.ErrNeedsNewConnect):
				// The plugin changed: it is shown to the user first, then the context is connected
				// anew (D-26; the section's approval, as a drill into it shows it, D-57).
				b.sh.kc.reviewPlugin(name, func(allowed bool) {
					if allowed {
						b.sh.dispatch("cluster.connect", map[string]any{"context": name}, nil)
					}
				})
			case err != nil:
				b.sh.showError(err)
			}
		})
	}
}

// startTicker redraws the countdown every second while the banner shows.
func (b *connBanner) startTicker() {
	if b.ticker != nil {
		return
	}
	b.ticker = time.NewTicker(time.Second)
	b.stop = make(chan struct{})
	t, stop := b.ticker, b.stop
	go func() {
		for {
			select {
			case <-stop:
				return
			case <-b.sh.stopTickers:
				return
			case <-t.C:
				b.sh.do(func() {
					if b.root.Visible() {
						b.fill(b.sh.ui.Connection)
					}
				})
			}
		}
	}()
}

func (b *connBanner) hide() {
	if b.ticker != nil {
		b.ticker.Stop()
		close(b.stop)
		b.ticker, b.stop = nil, nil
	}
	if b.root.Visible() {
		b.root.Hide()
		b.root.Refresh()
	}
}
