package engine

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
)

// olderT0 is when the test logs start; line i is written i seconds later.
var olderT0 = time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)

func olderAt(i int) time.Time { return olderT0.Add(time.Duration(i) * time.Second) }

// olderLog fills a container's log with n lines, "line 0" to "line n-1", one a second.
func olderLog(f *serverLogs, key string, n int) {
	for i := range n {
		f.add(key, olderAt(i), fmt.Sprintf("line %d", i))
	}
}

func stampedText(at time.Time, text string) string { return at.Format(time.RFC3339Nano) + " " + text }

// TestOlderLogsOfAPod: older lines of a pod's tab come from asking for the tail it holds plus a
// step again, keeping what is older than its oldest line (and the lines at that timestamp, which
// the tab drops if it holds them); the next request reaches the log's start and says the tab
// holds it whole; the room bounds what comes, its newest lines kept.
func TestOlderLogsOfAPod(t *testing.T) {
	s, _ := logsSession(t)
	f := newServerLogs()
	olderLog(f, "web-1-a/app", 2500)
	tab := openPodLogsTab(s, "web-1-a", "app")
	ctx := context.Background()

	// The tab holds the last 1,000 lines.
	got, err := s.olderLogs(ctx, tab, f, OlderRequest{Held: []HeldLog{{Lines: 1000, Oldest: olderAt(1500)}}, Room: 9000})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Lines) != 1001 || got.Lines[0] != stampedText(olderAt(500), "line 500") || got.Lines[1000] != stampedText(olderAt(1500), "line 1500") ||
		got.Complete || len(got.Whole) != 0 {
		t.Errorf("first step: %d lines from %q to %q, complete %v, whole %v", len(got.Lines), got.Lines[0], got.Lines[len(got.Lines)-1], got.Complete, got.Whole)
	}
	reqs := f.requestsFor("web-1-a/app")
	if r := reqs[len(reqs)-1]; r.TailLines != 2000 || r.Follow || !r.Timestamps || !r.SinceTime.IsZero() || r.Previous {
		t.Errorf("request %+v: want the 1,000 held and 1,000 more, once", r)
	}

	// Holding 2,000: the next step reaches the start.
	got, err = s.olderLogs(ctx, tab, f, OlderRequest{Held: []HeldLog{{Lines: 2000, Oldest: olderAt(500)}}, Room: 8000})
	if err != nil || len(got.Lines) != 501 || got.Lines[0] != stampedText(olderAt(0), "line 0") || !got.Complete || len(got.Whole) != 1 || got.Whole[0] != "" {
		t.Errorf("the start: %d lines, complete %v, whole %q, %v", len(got.Lines), got.Complete, got.Whole, err)
	}
	// Held whole: nothing is asked.
	n := len(f.requestsFor("web-1-a/app"))
	got, err = s.olderLogs(ctx, tab, f, OlderRequest{Held: []HeldLog{{Lines: 2500, Oldest: olderAt(0), Whole: true}}, Room: 7500})
	if err != nil || len(got.Lines) != 0 || !got.Complete || len(f.requestsFor("web-1-a/app")) != n {
		t.Errorf("a whole log asked again: %+v, %v", got, err)
	}

	// Room for 300: the 300 lines next to the held ones.
	got, err = s.olderLogs(ctx, tab, f, OlderRequest{Held: []HeldLog{{Lines: 1000, Oldest: olderAt(1500)}}, Room: 300})
	if err != nil || len(got.Lines) != 300 || got.Lines[0] != stampedText(olderAt(1201), "line 1201") || got.Complete {
		t.Errorf("bounded by the room: %d lines from %q, complete %v, %v", len(got.Lines), got.Lines[0], got.Complete, err)
	}
	if got, _ := s.olderLogs(ctx, tab, f, OlderRequest{Held: []HeldLog{{Lines: 10000, Oldest: olderAt(0)}}}); len(got.Lines) != 0 {
		t.Error("a full tab got lines")
	}
}

// TestOlderLogsOfAWorkload: a controlling resource's tab asks every container it shows that it
// does not hold whole, sharing the step; the lines come merged in time order with their prefixes;
// a container the server refuses is a note; the selection decides which containers are asked.
func TestOlderLogsOfAWorkload(t *testing.T) {
	s, _ := logsSession(t)
	f := newServerLogs()
	for i := range 300 {
		f.add("web-1-a/app", olderAt(2*i), fmt.Sprintf("a %d", i))
		f.add("web-1-b/app", olderAt(2*i+1), fmt.Sprintf("b %d", i))
	}
	f.add("web-1-a/proxy", olderAt(0), "proxy up")
	ctx := context.Background()
	tab, _, err := s.OpenWorkloadLogsTab(ctx, model.ObjectRef{Kind: "deployments", Namespace: "shop", Name: "web"})
	if err != nil {
		t.Fatal(err)
	}
	held := []HeldLog{
		{Source: "web-1-a/app", Lines: 100, Oldest: olderAt(400)},
		{Source: "web-1-b/app", Lines: 100, Oldest: olderAt(401)},
		{Source: "web-1-a/proxy", Lines: 1, Oldest: olderAt(0), Whole: true},
	}
	got, err := s.olderLogs(ctx, tab, f, OlderRequest{Held: held, Room: 9900})
	if err != nil {
		t.Fatal(err)
	}
	// 500 lines asked of each (100 held + 1,000 shared by two): all 300, of which 201 each are
	// older or at the oldest.
	if len(got.Lines) != 402 || got.Lines[0] != "web-1-a/app "+stampedText(olderAt(0), "a 0") ||
		got.Lines[1] != "web-1-b/app "+stampedText(olderAt(1), "b 0") || !got.Complete || len(got.Whole) != 2 {
		t.Errorf("%d lines from %q, complete %v, whole %v", len(got.Lines), got.Lines[:2], got.Complete, got.Whole)
	}
	for i := 1; i < len(got.Lines); i++ {
		if strings.SplitN(got.Lines[i], " ", 3)[1] < strings.SplitN(got.Lines[i-1], " ", 3)[1] {
			t.Fatalf("not in time order at %d: %q after %q", i, got.Lines[i], got.Lines[i-1])
		}
	}
	if len(f.requestsFor("web-1-a/proxy")) != 0 {
		t.Error("a container held whole was asked")
	}
	if r := f.requestsFor("web-1-b/app"); r[len(r)-1].TailLines != 600 {
		t.Errorf("tail %d, want 100 held + 500", r[len(r)-1].TailLines)
	}

	// Narrowed to web-1-b: only its container is asked.
	if _, err := s.SetLogSource(ctx, tab.ID, strp("web-1-b"), nil); err != nil {
		t.Fatal(err)
	}
	tab, _ = s.LogsTab(tab.ID)
	before := len(f.requestsFor("web-1-a/app"))
	if _, err := s.olderLogs(ctx, tab, f, OlderRequest{Held: held, Room: 9900}); err != nil {
		t.Fatal(err)
	}
	if len(f.requestsFor("web-1-a/app")) != before {
		t.Error("a container the tab does not show was asked")
	}
}

// TestOlderLogsNotes: a container whose log the server refuses becomes a note without a
// credential's shape changed, and the tab is not complete.
func TestOlderLogsNotes(t *testing.T) {
	s, _ := logsSession(t)
	f := newFakeLogs()
	f.fail["web-1-a/app"] = fmt.Errorf("pods \"web-1-a\" is forbidden:\nno")
	tab := openPodLogsTab(s, "web-1-a", "app")
	got, err := s.olderLogs(context.Background(), tab, f, OlderRequest{Held: []HeldLog{{Lines: 10, Oldest: olderAt(10)}}, Room: 100})
	if err != nil || got.Complete || len(got.Notes) != 1 || got.Notes[0] != NoticePrefix+` web-1-a/app: pods "web-1-a" is forbidden: no` {
		t.Errorf("%+v, %v", got, err)
	}
}

// TestRequestOlderLogsCounts: logs.loadOlder's engine side counts the requests on the tab.
func TestRequestOlderLogsCounts(t *testing.T) {
	s, _ := logsSession(t)
	tab := openPodLogsTab(s, "web-1-a", "app")
	for range 2 {
		if _, err := s.RequestOlderLogs(tab.ID); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := s.LogsTab(tab.ID); got.Logs.LoadOlder != 2 {
		t.Errorf("loadOlder %d", got.Logs.LoadOlder)
	}
	if _, err := s.RequestOlderLogs("logs-404"); err == nil {
		t.Error("a tab that does not exist")
	}
}

// TestReadOlderBoundsBytes: long lines cannot make a request for older lines keep more than its
// byte bound: the newest that fit stay, and the source is not held whole.
func TestReadOlderBoundsBytes(t *testing.T) {
	f := newServerLogs()
	long := strings.Repeat("x", 100)
	for i := range 10 {
		f.add("web-1-a/app", olderAt(i), fmt.Sprintf("%d %s", i, long))
	}
	opts := cluster.LogOptions{Namespace: "shop", Pod: "web-1-a", Container: "app", Timestamps: true, TailLines: 50}
	lineBytes := len(stampedText(olderAt(0), "0 "+long))
	got, n, trimmed, err := readOlder(context.Background(), f, opts, time.Time{}, 3*lineBytes+10)
	if err != nil || n != 10 || !trimmed || len(got) != 3 || !strings.HasPrefix(got[0].line, stampedText(olderAt(7), "7 ")) {
		t.Errorf("%d lines from %d, trimmed %v, %v", len(got), n, trimmed, err)
	}
	got, _, trimmed, err = readOlder(context.Background(), f, opts, time.Time{}, 100*lineBytes)
	if err != nil || trimmed || len(got) != 10 {
		t.Errorf("unbounded: %d lines, trimmed %v, %v", len(got), trimmed, err)
	}
}
