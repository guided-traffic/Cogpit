package engine

import (
	"strings"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// The connect log (G2): what happened when the user last connected to a context, line by line,
// as Lens shows it behind a warning icon beside the context ("Connection Updates", INFO and ERROR
// lines, captured 2026-10-12). Cogpit keeps the last attempt of each context in memory, for the
// session; the picker marks a context whose last attempt failed (ContextInfo.ConnectProblem) and
// cluster.connectLog returns the lines.
//
// A line carries only what the connect errors already carry, which never contain a credential
// (cluster.ConnectError, kubeconfig.SafeError); each is escaped for control characters and capped.
// What a credential plugin writes to its standard error is not in the log: client-go sends it to
// Cogpit's own standard error (gap G2).

// Levels of a connect log line: INFO and ERROR as Lens writes them; WARN is Cogpit's own, for a
// connection that works but is not safe (tlswarning.go), and does not mark a connect problem.
const (
	LogInfo  = "INFO"
	LogWarn  = "WARN"
	LogError = "ERROR"
)

const (
	maxConnectLogLines    = 100 // per context
	maxConnectLogContexts = 64  // contexts remembered
	maxConnectLogText     = 600 // bytes per line
)

// ConnectLogLine is one line of a context's connect log.
type ConnectLogLine struct {
	Time  time.Time `json:"time"`
	Level string    `json:"level"` // LogInfo, LogWarn or LogError
	Text  string    `json:"text"`
}

// connectLogs holds the logs by context id. The zero value is ready to use.
type connectLogs struct {
	mu    sync.Mutex
	byCtx map[string][]ConnectLogLine
}

// restart begins a new log for ctx with one line.
func (l *connectLogs) restart(ctx string, text string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.byCtx == nil {
		l.byCtx = map[string][]ConnectLogLine{}
	}
	if _, known := l.byCtx[ctx]; !known && len(l.byCtx) >= maxConnectLogContexts {
		for k := range l.byCtx { // forget one other context; which one does not matter
			delete(l.byCtx, k)
			break
		}
	}
	l.byCtx[ctx] = nil
	l.addLocked(ctx, LogInfo, text)
}

// add appends the lines of text (one per line of text) to ctx's log.
func (l *connectLogs) add(ctx, level, text string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.byCtx == nil {
		l.byCtx = map[string][]ConnectLogLine{}
	}
	l.addLocked(ctx, level, text)
}

func (l *connectLogs) addLocked(ctx, level, text string) {
	now := time.Now().UTC()
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		list := append(l.byCtx[ctx], ConnectLogLine{Time: now, Level: level, Text: kubeconfig.DisplayText(line, maxConnectLogText)})
		if len(list) > maxConnectLogLines {
			list = list[len(list)-maxConnectLogLines:]
		}
		l.byCtx[ctx] = list
	}
}

// lines returns a copy of ctx's log.
func (l *connectLogs) lines(ctx string) []ConnectLogLine {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]ConnectLogLine{}, l.byCtx[ctx]...)
}

// problem reports whether ctx's log ends in an error line.
func (l *connectLogs) problem(ctx string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	list := l.byCtx[ctx]
	return len(list) > 0 && list[len(list)-1].Level == LogError
}

// ConnectLog returns the log of the last attempt to connect to the context id (empty when there
// was none this session).
func (s *Session) ConnectLog(ctx string) []ConnectLogLine { return s.state.logs.lines(ctx) }

// markConnectProblems sets ConnectProblem on the contexts whose last attempt failed.
func (s *Session) markConnectProblems(list []ContextInfo) {
	for i := range list {
		list[i].ConnectProblem = s.state.logs.problem(list[i].Name)
	}
}
