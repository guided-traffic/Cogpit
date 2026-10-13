package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Jumping to a time in a logs tab (ui-parity › Logs, "Jump to a time and date", gap G8). Lens's
// log options offer to display the required time and date in the log (its documentation's "Log
// options"); how its control reads a time is not documented. Cogpit's tab has a text field (Fyne
// has no date picker) whose text the engine reads (ParseLogTime) and keeps on the tab
// (LogOptions.Jump), so that agents jump the same way; the window then scrolls to the first line
// written at or after it, loading older lines first when it holds none that old (logsolder.go).

// LogJump is a logs tab's latest jump to a time.
type LogJump struct {
	// Time is the time jumped to, in UTC.
	Time time.Time `json:"time"`
	// Seq counts the jumps, so that a jump to the same time scrolls there again.
	Seq int `json:"seq"`
}

// IsZero reports whether no jump was asked for (encoding/json's omitzero).
func (j LogJump) IsZero() bool { return j.Seq == 0 }

// errLogTime explains the forms ParseLogTime reads; the error names examples in the user's
// format besides (logTimeError).
var errLogTime = errors.New(`"10m ago", "1h30m ago", "2d ago", "now", ISO 8601 ("2026-10-13 01:20") ` +
	`or RFC 3339 ("2026-10-13T01:20:00Z")`)

// zonedLayouts are the ISO forms read in the time zone of the clock ParseLogTime is given, besides
// the user's format.
var zonedLayouts = []string{
	"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02 15:04",
	"2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02",
}

// ParseLogTime is ParseLogTimeIn with ISO 8601 as the user's format.
func ParseLogTime(text string, now time.Time) (time.Time, error) {
	return ParseLogTimeIn(text, now, model.TimeLayout{})
}

// ParseLogTimeIn reads the time a logs tab jumps to, relative to now and in now's time zone: a
// date with a time of day in the user's format (layout, D-31: "13.10.2026 01:20", seconds
// optional) or ISO 8601 ("2026-10-13 01:20"), or a date alone (midnight); a time of day alone
// ("01:20"), today, or yesterday when that is still to come (the logs hold the past; Cogpit's
// default); RFC 3339; a duration ago ("10m ago", "1h30m ago", "2d ago", "ago" optional); or
// "now".
func ParseLogTimeIn(text string, now time.Time, layout model.TimeLayout) (time.Time, error) {
	s := strings.TrimSpace(text)
	if s == "" {
		return time.Time{}, logTimeError("no time given", now, layout)
	}
	if strings.EqualFold(s, "now") {
		return now, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	loc := now.Location()
	for _, l := range append(layout.DateTimeLayouts(), zonedLayouts...) {
		if t, ok := parseTyped(l, s, loc); ok {
			return t, nil
		}
	}
	for _, l := range append(layout.TimeOfDayLayouts(), "15:04:05", "15:04") {
		if c, ok := parseTyped(l, s, loc); ok {
			y, m, d := now.Date()
			t := time.Date(y, m, d, c.Hour(), c.Minute(), c.Second(), c.Nanosecond(), loc)
			if t.After(now) {
				t = t.AddDate(0, 0, -1)
			}
			return t, nil
		}
	}
	if d, ok := parseAgo(s); ok {
		return now.Add(-d), nil
	}
	return time.Time{}, logTimeError(fmt.Sprintf("%q is not a time", s), now, layout)
}

// parseTyped reads s in a layout and zone as a user types it: spaces of any width as one space,
// and "pm" or "Pm" as "PM" where the layout writes "PM".
func parseTyped(layout, s string, loc *time.Location) (time.Time, bool) {
	s = strings.Map(func(r rune) rune {
		if r == '\u00a0' || r == '\u202f' {
			return ' '
		}
		return r
	}, s)
	if t, err := time.ParseInLocation(layout, s, loc); err == nil {
		return t, true
	}
	if strings.Contains(layout, "PM") {
		if t, err := time.ParseInLocation(layout, strings.ToUpper(s), loc); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// logTimeError says what ParseLogTime reads, with examples in the user's format; it wraps
// errLogTime.
func logTimeError(what string, now time.Time, layout model.TimeLayout) error {
	f := layout.In(now.Location())
	return fmt.Errorf("%s: give a time like %q, %q (today), %w", what,
		f.FormatDate(now)+" "+f.FormatClock(now), f.FormatClock(now), errLogTime)
}

// parseAgo reads "10m ago", "1h30m", "2d ago": a duration in Go's form, days allowed.
func parseAgo(s string) (time.Duration, bool) {
	s = strings.TrimSpace(strings.TrimSuffix(strings.ToLower(s), "ago"))
	if s == "" {
		return 0, false
	}
	var days time.Duration
	if i := strings.IndexByte(s, 'd'); i > 0 {
		n, err := strconv.Atoi(s[:i])
		if err != nil || n < 0 {
			return 0, false
		}
		days, s = time.Duration(n)*24*time.Hour, s[i+1:]
	}
	if s == "" {
		return days, true
	}
	d, err := time.ParseDuration(s)
	if err != nil || d < 0 {
		return 0, false
	}
	return days + d, true
}

// JumpLogs makes an open logs tab jump to the time text names (ParseLogTimeIn, by the session's
// clock, in the user's zone and format, which the tab shows its timestamps in, D-31): the window
// scrolls to the first line written at or after it. It returns the tab.
func (s *Session) JumpLogs(id, text string) (DockTab, error) {
	at, err := ParseLogTimeIn(text, s.displayNow(), s.opts.TimeLayout)
	if err != nil {
		return DockTab{}, err
	}
	return s.updateLogsTab(id, func(t *DockTab) {
		t.Logs.Jump = LogJump{Time: at.UTC(), Seq: t.Logs.Jump.Seq + 1}
	})
}
