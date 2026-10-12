package engine

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
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

// errLogTime explains the forms ParseLogTime reads.
var errLogTime = errors.New(`a time like "2026-10-13 01:20", "2026-10-13 01:20:30", "01:20" (today), ` +
	`"10m ago", "1h30m ago", "2d ago", "now" or RFC 3339 ("2026-10-13T01:20:00Z")`)

// zonedLayouts are the forms read in the time zone of the clock ParseLogTime is given.
var zonedLayouts = []string{
	"2006-01-02 15:04:05.999999999", "2006-01-02 15:04:05", "2006-01-02 15:04",
	"2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05", "2006-01-02T15:04", "2006-01-02",
}

// ParseLogTime reads the time a logs tab jumps to, relative to now and in now's time zone:
// RFC 3339; a date with a time ("2026-10-13 01:20", seconds optional) or a date alone
// (midnight); a time of day alone ("01:20"), today, or yesterday when that is still to come (the
// logs hold the past; Cogpit's default); a duration ago ("10m ago", "1h30m ago", "2d ago", "ago"
// optional); or "now".
func ParseLogTime(text string, now time.Time) (time.Time, error) {
	s := strings.TrimSpace(text)
	if s == "" {
		return time.Time{}, fmt.Errorf("no time given: %w", errLogTime)
	}
	if strings.EqualFold(s, "now") {
		return now, nil
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	loc := now.Location()
	for _, layout := range zonedLayouts {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, nil
		}
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		if c, err := time.ParseInLocation(layout, s, loc); err == nil {
			y, m, d := now.Date()
			t := time.Date(y, m, d, c.Hour(), c.Minute(), c.Second(), 0, loc)
			if t.After(now) {
				t = t.AddDate(0, 0, -1)
			}
			return t, nil
		}
	}
	if d, ok := parseAgo(s); ok {
		return now.Add(-d), nil
	}
	return time.Time{}, fmt.Errorf("%q is not %w", s, errLogTime)
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

// JumpLogs makes an open logs tab jump to the time text names (ParseLogTime, by the session's
// clock, in UTC: the zone of the timestamps the tab shows, Cogpit's default): the window scrolls
// to the first line written at or after it. It returns the tab.
func (s *Session) JumpLogs(id, text string) (DockTab, error) {
	at, err := ParseLogTime(text, s.Now().UTC())
	if err != nil {
		return DockTab{}, err
	}
	return s.updateLogsTab(id, func(t *DockTab) {
		t.Logs.Jump = LogJump{Time: at.UTC(), Seq: t.Logs.Jump.Seq + 1}
	})
}
