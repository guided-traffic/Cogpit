package fyneui

import (
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/ui/fyneui/yamlview"
)

const (
	perfFrame = 16 * time.Millisecond // one frame at 60 Hz
	perfGrace = 10 * time.Second      // how long the meter waits for rows before it starts anyway
	perfRows  = 3                     // rows the auto-scroll advances per frame
)

// Perf modes (Options.PerfMode).
const (
	// PerfScroll measures while auto-scrolling the open table for PerfDuration (gate S1).
	PerfScroll = "scroll"
	// PerfGate runs every gate measurement in one window session: start-up, scrolling, a switch
	// through every view, and typing into a 5 MB YAML document (gates S1 and S4).
	PerfGate = "gate"
)

// Phase durations of PerfGate; variables so that tests can shorten them.
var (
	gateScrollFor = 10 * time.Second
	gateViewDwell = 3 * time.Second
	gateYAMLFor   = 10 * time.Second
	gateYAMLBytes = 5 << 20
)

// switchWindow is how long after a view switch its stalls count toward the switch.
const switchWindow = time.Second

// perfMeter is the jank meter for the gate (docs/developer/spike-fyne.md). Fyne has no frame
// callback, so a ticker posts a timestamped function to the main goroutine every frame and the
// meter records how late it runs: a callback that waits 30 ms waited for a frame of 30 ms. It
// records from the moment the shell starts, in phases: "startup" until the table first has rows,
// then "scroll" (the table scrolls through its rows), and in PerfGate mode "views" (every view
// opened in turn) and "yaml" (a keystroke per frame into a 5 MB document), then reports and quits.
type perfMeter struct {
	sh     *shell
	begin  time.Time
	mode   string
	report sync.Once
	// stop ends the ticker once the meter has reported: a finished measurement must not keep
	// posting to a main goroutine that may be slower than one callback per frame (a loaded CI
	// runner with the race detector), which would then never catch up.
	stop chan struct{}

	// The rest is main goroutine only.
	samples  []perfSample
	phases   []perfPhaseMark
	scrollY  float32
	switches []PerfSwitch
	views    []string // views still to open in the views phase
	nextAt   time.Time

	yamlDoc   chan string // the generated document, built off the main goroutine
	yamlText  string
	yamlWin   fyne.Window
	yamlView  *yamlview.View
	keys      PerfKeys
	keyWork   []time.Duration
	yamlStart time.Time
}

// perfSample is one callback: when it was posted, relative to the meter's begin, and how late it
// ran.
type perfSample struct {
	at, late time.Duration
}

// perfPhaseMark is where a phase starts.
type perfPhaseMark struct {
	name string
	at   time.Duration
}

// startPerf starts the meter; call it once, before the event loop runs.
func (sh *shell) startPerf() {
	m := &perfMeter{sh: sh, begin: time.Now(), mode: sh.opts.PerfMode, stop: make(chan struct{})}
	if m.mode == "" {
		m.mode = PerfScroll
	}
	m.phases = []perfPhaseMark{{name: "startup"}}
	if m.mode == PerfGate {
		m.yamlDoc = make(chan string, 1)
		go func() { m.yamlDoc <- perfYAML(gateYAMLBytes) }()
	}
	sh.perf = m
	go m.run()
}

func (m *perfMeter) run() {
	t := time.NewTicker(perfFrame)
	defer t.Stop()
	for {
		select {
		case <-m.sh.stopTickers:
			return
		case <-m.stop:
			return
		case <-t.C:
			posted := time.Now()
			m.sh.do(func() { m.sample(posted) })
		}
	}
}

func (m *perfMeter) phase() string { return m.phases[len(m.phases)-1].name }

func (m *perfMeter) enter(name string) {
	m.phases = append(m.phases, perfPhaseMark{name: name, at: time.Since(m.begin)})
}

// sample records one callback's lateness and moves the phases on; on the main goroutine. A sample
// still queued when the meter reported does nothing: it would otherwise end the measurement a
// second time and close the YAML window twice (Fyne's test driver panics on that while holding its
// window lock, and every later call into the driver then blocks).
func (m *perfMeter) sample(posted time.Time) {
	select {
	case <-m.stop:
		return
	default:
	}
	m.samples = append(m.samples, perfSample{at: posted.Sub(m.begin), late: time.Since(posted)})
	now := time.Now()
	tv := m.sh.view
	switch m.phase() {
	case "startup":
		if (tv.snap != nil && len(tv.snap.Rows) > 0) || time.Since(m.begin) >= perfGrace {
			m.enter("scroll")
		}
	case "scroll":
		m.scroll()
		d := m.sh.opts.PerfDuration
		if m.mode == PerfGate {
			d = gateScrollFor
		}
		if d <= 0 || time.Since(m.begin)-m.phases[len(m.phases)-1].at < d {
			return
		}
		if m.mode != PerfGate {
			m.done()
			return
		}
		m.enter("views")
		m.views = m.viewOrder()
		m.nextAt = now
	case "views":
		m.trackSwitch()
		if now.Before(m.nextAt) || m.awaitingRows(now) {
			return
		}
		if len(m.views) == 0 {
			m.enter("yaml")
			m.startYAML()
			return
		}
		v := m.views[0]
		m.views = m.views[1:]
		m.switches = append(m.switches, PerfSwitch{View: v, AtS: time.Since(m.begin).Seconds(), RowsAfterMs: -1})
		m.sh.dispatch("ui.openView", openViewParams{View: v}, nil)
		m.nextAt = now.Add(gateViewDwell)
	case "yaml":
		if m.yamlView == nil {
			m.startYAML() // the document was not built yet on the phase's first frame
			return
		}
		m.typeKey()
		if time.Since(m.yamlStart) >= gateYAMLFor {
			m.done()
		}
	}
}

// viewOrder is every view of the sidebar once, ending with the one open now, so that the last
// switch returns to a view whose informer is still warm (budget P10).
func (m *perfMeter) viewOrder() []string {
	cur := m.sh.ui.View
	var out []string
	for _, k := range m.sh.kinds {
		if k.Kind != cur {
			out = append(out, k.Kind)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

// awaitingRows reports whether the last switched-to view has shown no rows yet and may still: the
// next switch waits for them, at most perfGrace after the switch, so that a slow machine measures
// every view instead of switching away before its rows arrive. A fast one never waits here: its
// rows are on screen long before the dwell time ends.
func (m *perfMeter) awaitingRows(now time.Time) bool {
	if len(m.switches) == 0 {
		return false
	}
	sw := m.switches[len(m.switches)-1]
	return sw.RowsAfterMs < 0 && now.Sub(m.begin) < time.Duration(sw.AtS*float64(time.Second))+perfGrace
}

// trackSwitch records when the last switched-to view first shows rows.
func (m *perfMeter) trackSwitch() {
	if len(m.switches) == 0 {
		return
	}
	sw := &m.switches[len(m.switches)-1]
	if sw.RowsAfterMs >= 0 {
		return
	}
	if snap := m.sh.view.snap; snap != nil && snap.View == sw.View && len(snap.Rows) > 0 {
		sw.RowsAfterMs = (time.Since(m.begin).Seconds() - sw.AtS) * 1000
		sw.Rows = len(snap.Rows)
	}
}

// scroll moves the table down by a few rows, and back to the top at the end.
func (m *perfMeter) scroll() {
	tv := m.sh.view
	if tv.snap == nil || len(tv.snap.Rows) == 0 || len(tv.cells) == 0 {
		return
	}
	th := tv.table.Theme()
	rowH := tv.cells[0].MinSize().Height + th.Size(theme.SizeNamePadding)
	m.scrollY += perfRows * rowH
	if m.scrollY > float32(len(tv.snap.Rows))*rowH-tv.table.Size().Height {
		m.scrollY = 0
	}
	tv.table.ScrollToOffset(fyne.NewPos(0, m.scrollY))
}

// startYAML opens a window with the editor on the generated document, caret in the middle. It
// never waits on the main goroutine: while the document is still being built it returns, and the
// next frame tries again.
func (m *perfMeter) startYAML() {
	select {
	case m.yamlText = <-m.yamlDoc:
	default:
		return
	}
	m.yamlView = yamlview.New()
	m.yamlWin = m.sh.app.NewWindow("Cogpit: YAML editor, 5 MB")
	m.yamlWin.SetContent(m.yamlView)
	m.yamlWin.Resize(fyne.NewSize(1000, 800))
	m.yamlWin.Show()
	t := time.Now()
	m.yamlView.SetText(m.yamlText)
	m.keys.SetTextMs = ms(time.Since(t))
	m.keys.DocBytes = len(m.yamlText)
	m.yamlView.SetReadOnly(false)
	m.yamlView.SetCursorPosition(m.yamlView.LineCount()/2, 4)
	m.yamlStart = time.Now()
}

// typeKey types one character, as a user typing fast would, and records the widget's own work.
func (m *perfMeter) typeKey() {
	if m.yamlView == nil {
		return
	}
	t := time.Now()
	m.yamlView.TypedRune('x')
	m.keyWork = append(m.keyWork, time.Since(t))
}

// done reports and quits.
func (m *perfMeter) done() {
	m.finish()
	if m.yamlWin != nil {
		m.yamlWin.Close()
	}
	m.sh.app.Quit()
}

// finish stops the ticker, computes the statistics and hands them to Options.PerfReport, once.
func (m *perfMeter) finish() {
	m.report.Do(func() {
		close(m.stop)
		if m.sh.opts.PerfReport != nil {
			m.sh.opts.PerfReport(m.stats())
		}
	})
}

// stats builds the report: the top-level fields cover the scroll phase, as in PerfScroll mode;
// Phases covers every phase, with the view switches and keystrokes.
func (m *perfMeter) stats() PerfStats {
	total := time.Since(m.begin)
	var out PerfStats
	phases := make([]PerfPhase, 0, len(m.phases))
	for i, ph := range m.phases {
		end := total
		if i+1 < len(m.phases) {
			end = m.phases[i+1].at
		}
		var in []perfSample
		for _, s := range m.samples {
			if s.at >= ph.at && s.at < end {
				in = append(in, perfSample{at: s.at - ph.at, late: s.late})
			}
		}
		st := perfStats(in, time.Time{})
		p := PerfPhase{
			Name: ph.name, FromS: ph.at.Seconds(), ToS: end.Seconds(),
			Samples: st.Samples, P50Ms: st.P50Ms, P99Ms: st.P99Ms, MaxMs: st.MaxMs, Over16Ms: st.Over16Ms, Stalls: st.Stalls,
		}
		switch ph.name {
		case "scroll":
			st.DurationS = (end - ph.at).Seconds()
			out = st
		case "views":
			p.Switches = m.switchStats()
		case "yaml":
			keys := m.keys
			keys.Count = len(m.keyWork)
			if len(m.keyWork) > 0 {
				sorted := slices.Clone(m.keyWork)
				slices.Sort(sorted)
				keys.WorkP50Ms = ms(sorted[len(sorted)/2])
				keys.WorkP99Ms = ms(sorted[min(len(sorted)-1, len(sorted)*99/100)])
				keys.WorkMaxMs = ms(sorted[len(sorted)-1])
			}
			p.Keystrokes = &keys
		}
		phases = append(phases, p)
	}
	out.Phases = phases
	return out
}

// switchStats adds to each view switch the longest callback wait within switchWindow after it.
func (m *perfMeter) switchStats() []PerfSwitch {
	out := slices.Clone(m.switches)
	for i := range out {
		from := time.Duration(out[i].AtS * float64(time.Second))
		for _, s := range m.samples {
			if s.at >= from && s.at < from+switchWindow {
				out[i].MaxLateMs = max(out[i].MaxLateMs, ms(s.late))
			}
		}
	}
	return out
}

func ms(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }

// maxStalls bounds PerfStats.Stalls.
const maxStalls = 20

// perfStats summarises callback latencies. Callbacks queue up behind a stall, so one long stall
// makes several late samples; Stalls lists each stall once (its first late sample, which waited
// longest), so that a stall at start-up can be told from steady jank.
func perfStats(samples []perfSample, start time.Time) PerfStats {
	s := PerfStats{Samples: len(samples)}
	if !start.IsZero() {
		s.DurationS = time.Since(start).Seconds()
	}
	if len(samples) == 0 {
		return s
	}
	sorted := make([]time.Duration, len(samples))
	for i, x := range samples {
		sorted[i] = x.late
	}
	slices.Sort(sorted)
	at := func(q float64) time.Duration { return sorted[min(len(sorted)-1, int(q*float64(len(sorted))))] }
	s.P50Ms, s.P99Ms, s.MaxMs = ms(at(0.50)), ms(at(0.99)), ms(sorted[len(sorted)-1])
	inStall := false
	for _, x := range samples {
		if x.late <= perfFrame {
			inStall = false
			continue
		}
		s.Over16Ms++
		if !inStall && len(s.Stalls) < maxStalls {
			s.Stalls = append(s.Stalls, PerfStall{AtS: x.at.Seconds(), Ms: ms(x.late)})
		}
		inStall = true
	}
	return s
}

// perfYAML builds a synthetic multi-document YAML of at least size bytes: pod-like objects with
// keys, quoted and plain strings, numbers, lists and comments, for the editor measurement.
func perfYAML(size int) string {
	var b strings.Builder
	b.Grow(size + 1024)
	for i := 0; b.Len() < size; i++ {
		fmt.Fprintf(&b, `---
apiVersion: v1
kind: Pod
metadata:
  name: web-%d
  namespace: demo-%d
  labels:
    app: web
    tier: "frontend"
  annotations:
    note: "generated, # not a comment"
spec:
  containers:
    - name: app
      image: registry.example/app:1.%d
      ports:
        - containerPort: 8080
          protocol: TCP
      env:
        - name: MODE
          value: production
  restartPolicy: Always # a comment
status:
  phase: Running
  podIP: 10.244.%d.%d
`, i, i%20, i%9, (i/250)%250, i%250)
	}
	return b.String()
}
