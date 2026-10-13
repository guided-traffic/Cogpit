package engine

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/model"
)

// TestParseLogTime: the forms a jump reads, in the clock's time zone; a time of day still to
// come is yesterday's; anything else says what it reads.
func TestParseLogTime(t *testing.T) {
	berlin := time.FixedZone("CEST", 2*60*60)
	now := time.Date(2026, 10, 13, 1, 30, 0, 0, berlin)
	for in, want := range map[string]time.Time{
		"2026-10-13 01:20":              time.Date(2026, 10, 13, 1, 20, 0, 0, berlin),
		" 2026-10-12 23:59:30 ":         time.Date(2026, 10, 12, 23, 59, 30, 0, berlin),
		"2026-10-13T01:20":              time.Date(2026, 10, 13, 1, 20, 0, 0, berlin),
		"2026-10-12":                    time.Date(2026, 10, 12, 0, 0, 0, 0, berlin),
		"01:20":                         time.Date(2026, 10, 13, 1, 20, 0, 0, berlin),
		"01:20:15":                      time.Date(2026, 10, 13, 1, 20, 15, 0, berlin),
		"23:00":                         time.Date(2026, 10, 12, 23, 0, 0, 0, berlin), // still to come today: yesterday
		"10m ago":                       now.Add(-10 * time.Minute),
		"1h30m ago":                     now.Add(-90 * time.Minute),
		"2d ago":                        now.Add(-48 * time.Hour),
		"1d2h":                          now.Add(-26 * time.Hour),
		"90s":                           now.Add(-90 * time.Second),
		"now":                           now,
		"2026-10-12T22:00:00Z":          time.Date(2026, 10, 12, 22, 0, 0, 0, time.UTC),
		"2026-10-12T22:00:00.5+02:00":   time.Date(2026, 10, 12, 22, 0, 0, 5e8, berlin),
		"2026-10-12 22:00:00.123456789": time.Date(2026, 10, 12, 22, 0, 0, 123456789, berlin),
	} {
		got, err := ParseLogTime(in, now)
		if err != nil || !got.Equal(want) {
			t.Errorf("%q: %v, %v; want %v", in, got, err, want)
		}
	}
	for _, in := range []string{"", "yesterday", "-5m", "13.10.2026", "25:00", "10 minutes ago", "d"} {
		if _, err := ParseLogTime(in, now); !errors.Is(err, errLogTime) {
			t.Errorf("%q: %v, want the forms explained", in, err)
		}
	}
}

// TestParseLogTimeInTheUsersFormat: a jump reads the format the tab's timestamps show (D-31), with
// or without seconds, a date alone and a time of day, besides ISO 8601; "pm" in any case where the
// format writes PM; an error names examples in the user's format.
func TestParseLogTimeInTheUsersFormat(t *testing.T) {
	berlin := time.FixedZone("CEST", 2*60*60)
	now := time.Date(2026, 10, 13, 13, 30, 0, 0, berlin)
	de := model.TimeLayout{Date: "02.01.2006", Time: "15:04:05"}
	us := model.TimeLayout{Date: "1/2/2006", Time: "3:04:05 PM"}
	for _, tc := range []struct {
		layout model.TimeLayout
		in     string
		want   time.Time
	}{
		{de, "13.10.2026 01:20:15", time.Date(2026, 10, 13, 1, 20, 15, 0, berlin)},
		{de, "13.10.2026 01:20", time.Date(2026, 10, 13, 1, 20, 0, 0, berlin)},
		{de, "12.10.2026", time.Date(2026, 10, 12, 0, 0, 0, 0, berlin)},
		{de, "13.10.2026 01:20:15.250", time.Date(2026, 10, 13, 1, 20, 15, 25e7, berlin)}, // as a log line shows it
		{de, "2026-10-13 01:20", time.Date(2026, 10, 13, 1, 20, 0, 0, berlin)},            // ISO still
		{de, "13:00", time.Date(2026, 10, 13, 13, 0, 0, 0, berlin)},
		{us, "10/13/2026 1:20:15 PM", time.Date(2026, 10, 13, 13, 20, 15, 0, berlin)},
		{us, "10/13/2026 1:20 pm", time.Date(2026, 10, 13, 13, 20, 0, 0, berlin)},
		{us, "10/13/2026\u202f1:20\u202fAM", time.Date(2026, 10, 13, 1, 20, 0, 0, berlin)},
		{us, "2:00 PM", time.Date(2026, 10, 12, 14, 0, 0, 0, berlin)}, // still to come today: yesterday
		{us, "10m ago", now.Add(-10 * time.Minute)},
	} {
		got, err := ParseLogTimeIn(tc.in, now, tc.layout)
		if err != nil || !got.Equal(tc.want) {
			t.Errorf("%q in %s: %v, %v; want %v", tc.in, tc.layout.Date, got, err, tc.want)
		}
	}
	_, err := ParseLogTimeIn("soon", now, de)
	if !errors.Is(err, errLogTime) || !strings.Contains(err.Error(), `"13.10.2026 13:30"`) || !strings.Contains(err.Error(), `"13:30" (today)`) {
		t.Errorf("the error: %v", err)
	}
}

// TestJumpLogsInTheUsersZone: logs.jumpTo reads a time in the session's format and the
// Preferences' zone, as the tab shows its timestamps, and keeps it in UTC.
func TestJumpLogsInTheUsersZone(t *testing.T) {
	s, _ := logsSessionWith(t, Options{Preferences: NewPreferencesStore(""),
		TimeLayout: model.TimeLayout{Date: "02.01.2006", Time: "15:04:05"}})
	if _, err := s.ChangePreferences(func(p *Preferences) error { p.Timezone = "Europe/Berlin"; return nil }); err != nil {
		t.Fatal(err)
	}
	tab := openPodLogsTab(s, "web-1-a", "app")
	got, err := s.JumpLogs(tab.ID, "13.10.2026 01:20")
	if err != nil || !got.Logs.Jump.Time.Equal(time.Date(2026, 10, 12, 23, 20, 0, 0, time.UTC)) {
		t.Fatalf("jump %+v, %v", got.Logs.Jump, err)
	}
}

// TestJumpLogs: a jump keeps the time in UTC and counts, so that the same time jumps again; a
// time the engine cannot read leaves the tab as it was.
func TestJumpLogs(t *testing.T) {
	s, _ := logsSession(t)
	tab := openPodLogsTab(s, "web-1-a", "app")
	got, err := s.JumpLogs(tab.ID, "2026-10-13T01:20:00+02:00")
	if err != nil || !got.Logs.Jump.Time.Equal(time.Date(2026, 10, 12, 23, 20, 0, 0, time.UTC)) || got.Logs.Jump.Time.Location() != time.UTC || got.Logs.Jump.Seq != 1 {
		t.Fatalf("jump %+v, %v", got.Logs.Jump, err)
	}
	if got, _ = s.JumpLogs(tab.ID, "2026-10-13T01:20:00+02:00"); got.Logs.Jump.Seq != 2 {
		t.Errorf("the same time again: %+v", got.Logs.Jump)
	}
	if _, err := s.JumpLogs(tab.ID, "soon"); err == nil {
		t.Error("an unreadable time")
	}
	if cur, _ := s.LogsTab(tab.ID); cur.Logs.Jump.Seq != 2 {
		t.Errorf("an unreadable time changed the tab: %+v", cur.Logs.Jump)
	}
}

// TestOlderLogsSince: a jump to a time older than the tab holds asks for the lines since then,
// bounded by the room (tailLines), and keeps what is older than the oldest held; a time the tab
// holds asks nothing; a time beyond the room's reach gets the newest lines that fit and is not
// complete.
func TestOlderLogsSince(t *testing.T) {
	s, _ := logsSession(t)
	f := newServerLogs()
	olderLog(f, "web-1-a/app", 2500)
	tab := openPodLogsTab(s, "web-1-a", "app")
	ctx := context.Background()
	held := []HeldLog{{Lines: 1000, Oldest: olderAt(1500)}}

	got, err := s.olderLogs(ctx, tab, f, OlderRequest{Held: held, Since: olderAt(1200).Add(500 * time.Millisecond), Room: 9000})
	if err != nil || len(got.Lines) != 301 || got.Lines[0] != stampedText(olderAt(1200), "line 1200") || !got.Complete || len(got.Whole) != 0 {
		t.Errorf("since 1200.5: %d lines from %q, complete %v, whole %v, %v", len(got.Lines), got.Lines[0], got.Complete, got.Whole, err)
	}
	reqs := f.requestsFor("web-1-a/app")
	if r := reqs[len(reqs)-1]; !r.SinceTime.Equal(olderAt(1200)) || r.TailLines != 10000 || r.Follow {
		t.Errorf("request %+v: want sinceTime in whole seconds, the room as the bound", r)
	}

	n := len(f.requestsFor("web-1-a/app"))
	got, err = s.olderLogs(ctx, tab, f, OlderRequest{Held: held, Since: olderAt(1600), Room: 9000})
	if err != nil || len(got.Lines) != 0 || !got.Complete || len(f.requestsFor("web-1-a/app")) != n {
		t.Errorf("a time the tab holds: %+v, %v", got, err)
	}

	got, err = s.olderLogs(ctx, tab, f, OlderRequest{Held: held, Since: olderAt(0), Room: 100})
	if err != nil || len(got.Lines) != 100 || got.Lines[0] != stampedText(olderAt(1401), "line 1401") || got.Complete {
		t.Errorf("beyond the room: %d lines from %q, complete %v, %v", len(got.Lines), got.Lines[0], got.Complete, err)
	}
}
