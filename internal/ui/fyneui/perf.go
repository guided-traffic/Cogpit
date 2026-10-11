package fyneui

import (
	"slices"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

const (
	perfFrame = 16 * time.Millisecond // one frame at 60 Hz
	perfGrace = 10 * time.Second      // how long the meter waits for rows before it starts anyway
	perfRows  = 3                     // rows the auto-scroll advances per frame
)

// perfMeter is the jank meter for gate S1 (docs/developer/spike-fyne.md). Fyne has no frame
// callback, so a ticker posts a timestamped function to the main goroutine every frame and the
// meter records how late it runs: a callback that waits 30 ms waited for a frame of 30 ms. With
// Options.PerfDuration it also scrolls the table through its rows, then reports and quits.
type perfMeter struct {
	sh      *shell
	samples []perfSample // main goroutine only
	begin   time.Time
	start   time.Time
	started bool
	scrollY float32
	report  sync.Once
}

// startPerf starts the meter; call it once, before the event loop runs.
func (sh *shell) startPerf() {
	m := &perfMeter{sh: sh, begin: time.Now()}
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
		case <-t.C:
			posted := time.Now()
			m.sh.do(func() { m.sample(posted) })
		}
	}
}

// sample records one callback's lateness; on the main goroutine.
func (m *perfMeter) sample(posted time.Time) {
	late := time.Since(posted)
	if !m.started {
		// Measure the shell at work: from the moment the table has rows, or after the grace
		// period if none arrive.
		if (m.sh.view.snap == nil || len(m.sh.view.snap.Rows) == 0) && time.Since(m.begin) < perfGrace {
			return
		}
		m.started, m.start = true, time.Now()
	}
	m.samples = append(m.samples, perfSample{at: time.Since(m.start), late: late})
	if d := m.sh.opts.PerfDuration; d > 0 {
		m.scroll()
		if time.Since(m.start) >= d {
			m.finish()
			m.sh.app.Quit()
		}
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

// finish computes the statistics and hands them to Options.PerfReport, once.
func (m *perfMeter) finish() {
	m.report.Do(func() {
		if m.sh.opts.PerfReport != nil {
			m.sh.opts.PerfReport(perfStats(m.samples, m.start))
		}
	})
}

// perfSample is one callback: when it was posted, relative to the meter's start, and how late it
// ran.
type perfSample struct {
	at, late time.Duration
}

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
	ms := func(d time.Duration) float64 { return float64(d) / float64(time.Millisecond) }
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
