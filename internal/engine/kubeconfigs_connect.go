package engine

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// KubeconfigSources returns the connector's kubeconfig sources; nil when it has none.
func (c *ClusterConnector) KubeconfigSources() *KubeconfigSources { return c.sources() }

// sources returns c.Sources bound to c.ClusterSettings, so that every login the sources describe
// binds the proxy this connector's connections go through (A23); nil without sources.
func (c *ClusterConnector) sources() *KubeconfigSources {
	if c.Sources != nil {
		c.Sources.UseClusterSettings(c.ClusterSettings)
	}
	return c.Sources
}

// withSourceContexts adds the sources' contexts to out. With sources every context comes from a
// source file read on its own (D-28): there is no merged default kubeconfig whose names a source
// id could hide.
func (c *ClusterConnector) withSourceContexts(out []ContextInfo, err error) ([]ContextInfo, error) {
	if c.Sources == nil {
		return out, err
	}
	extra, sErr := c.sources().Contexts()
	return append(out, extra...), errors.Join(err, sErr)
}

// ResolveContext turns what the user or an agent typed into a context id. An id of a source
// context ("dev [config]") and the demo cluster are returned as they are. A bare context name
// ("dev", as `--context` and kubectl know it) is resolved to the id of the one source file that
// defines it; defined in several files, it is refused with the ids to choose from. Without
// sources the name is returned unchanged.
func (c *ClusterConnector) ResolveContext(name string) (string, error) {
	if c.Sources == nil || (c.demoOffered() && name == DemoContext) {
		return name, nil
	}
	if _, _, ok, err := c.Sources.Find(name); err != nil || ok {
		return name, err
	}
	files, _, err := c.Sources.Files()
	if err != nil {
		return name, err
	}
	var ids []string
	for _, f := range files {
		for _, k := range f.Contexts {
			if k.Name == name {
				ids = append(ids, SourceContextID(k.Name, f.Label))
			}
		}
	}
	switch len(ids) {
	case 0:
		return name, nil
	case 1:
		return ids[0], nil
	}
	slices.Sort(ids)
	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = kubeconfig.DisplayText(id, 400)
	}
	return name, fmt.Errorf("the context %q is defined in several kubeconfig files; connect one of %s",
		kubeconfig.DisplayText(name, 200), strings.Join(quoted, ", "))
}

// contextResolver is a Connector that resolves a typed context name to an id (*ClusterConnector).
type contextResolver interface {
	ResolveContext(name string) (string, error)
}

// connectOptions returns the cluster options for a context id: a source context's file alone,
// with the sources' exec policy, and its id. Without sources it is the merged KubeconfigPaths
// and "". The policy binds no proxy of Cogpit's yet: Connect adds the one it connects through.
func (c *ClusterConnector) connectOptions(name string) (cluster.Options, string, error) {
	opts := cluster.Options{KubeconfigPaths: c.KubeconfigPaths, Context: name}
	if c.Sources == nil {
		return opts, "", nil
	}
	name, err := c.ResolveContext(name)
	if err != nil {
		return opts, "", err
	}
	f, kctx, ok, err := c.sources().Find(name)
	if err != nil {
		return opts, "", err
	}
	if !ok {
		return opts, "", &cluster.ConnectError{Context: name, Problem: "no such context in the kubeconfig sources (cluster_contexts lists them)"}
	}
	if f.Err != nil {
		return opts, "", f.Err
	}
	return cluster.Options{KubeconfigPaths: []string{f.Path}, Context: kctx, ExecPolicy: c.Sources.Policy}, name, nil
}

// sourcesConnector is a Connector that has kubeconfig sources (*ClusterConnector).
type sourcesConnector interface {
	KubeconfigSources() *KubeconfigSources
}

// KubeconfigSources returns the kubeconfig sources of the session's connector; nil when it has
// none (the demo-only and test connectors).
func (s *Session) KubeconfigSources() *KubeconfigSources {
	if c, ok := s.opts.Connector.(sourcesConnector); ok {
		return c.KubeconfigSources()
	}
	return nil
}

// ExecPluginInfo is what the user is shown before a source context's credential plugin first runs
// (kubeconfig.execPlugin): the plugin, whether it may run, and where its command is found now.
type ExecPluginInfo struct {
	// Context is the context id; Source the label of its file.
	Context string `json:"context"`
	Source  string `json:"source"`
	// State is ExecApproved, ExecRequired or ExecChanged.
	State string `json:"state"`
	// Kind is what needs the approval: kubeconfig.LoginProgram, LoginTokenFile or LoginBoth.
	Kind string `json:"kind"`
	// Fingerprint binds an approval to exactly this plugin (kubeconfig.approveExec takes it back, with the other contexts of its group).
	Fingerprint string                 `json:"fingerprint"`
	Plugin      *kubeconfig.ExecPlugin `json:"plugin"`
	// Found is where the command is found now: itself when it is a path, else the first match on
	// PATH; empty when it is not found.
	Found string `json:"found,omitempty"`
	// RiskyEnv are the variables the plugin's environment sets that change which program runs or
	// what it loads (PATH, LD_*, DYLD_*).
	RiskyEnv []string `json:"riskyEnv,omitempty"`
}

// ExecPlugin returns the credential plugin of a source context and whether it may run. A context
// that no source offers, or one without a plugin, is an error: no approval applies.
func (s *Session) ExecPlugin(id string) (*ExecPluginInfo, error) {
	k := s.KubeconfigSources()
	if k == nil {
		return nil, errors.New("this Cogpit has no kubeconfig sources")
	}
	f, kctx, ok, err := k.Find(id)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("%q is not a context of a kubeconfig source (cluster_contexts lists them)", id)
	}
	p := f.Exec[kctx]
	if p == nil {
		return nil, fmt.Errorf("the context %q runs no credential plugin", id)
	}
	state, err := k.ExecState(p)
	if err != nil {
		return nil, err
	}
	return &ExecPluginInfo{
		Context: id, Source: f.Label, State: state, Kind: p.Kind(), Fingerprint: p.Fingerprint(), Plugin: p,
		Found: cluster.LookPathNow(p.Command), RiskyEnv: p.RiskyEnv(),
	}, nil
}

// WatchKubeconfigs looks at the kubeconfig sources every interval (zero means
// DefaultKubeconfigPoll) and emits EventContexts when a source's files changed: a file appeared in
// or left a sync folder, or an added file changed. It compares the files' size
// and modification time and reads no content. It returns a function that stops watching; watching
// also ends when the session closes. Without sources it does nothing.
func (s *Session) WatchKubeconfigs(interval time.Duration) (stop func()) {
	k := s.KubeconfigSources()
	if k == nil {
		return func() {}
	}
	if interval <= 0 {
		interval = DefaultKubeconfigPoll
	}
	done := make(chan struct{})
	var once sync.Once
	stop = func() { once.Do(func() { close(done) }) }
	// Taken before returning, so that every change after WatchKubeconfigs returns is reported.
	last := k.signature()
	go func() {
		defer s.recoverTo("watching the kubeconfig sources")
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-done:
				return
			case <-t.C:
			}
			if s.closed() {
				return
			}
			if sig := k.signature(); sig != last {
				last = sig
				s.CloseGoneSourceConnection()
				s.closeTabsOfGoneFiles() // and the tabs of a context whose file went (persist_restore.go)
				s.emit(Event{Type: EventContexts})
			}
		}
	}()
	return stop
}

// CloseGoneSourceConnection closes the connection if it was made from a kubeconfig source's file
// and that file is gone, or its source was removed (Lens closes a cluster whose file leaves a
// synced folder; captured 2026-10-12). The UI is left Disconnected, with the reason in
// Connection.Error, a warning toast, and the reason in the context's connect log. It reports
// whether it closed a connection.
//
// Only a plain fact closes a connection: the file does not exist, or no source covers it. A file
// that is merely being rewritten, or does not parse for a moment, leaves the connection alone (it
// was built from the file as it was); so does a context dropped from a file that stays.
func (s *Session) CloseGoneSourceConnection() bool {
	k := s.KubeconfigSources()
	b := s.Backend()
	if k == nil || b == nil || b.SourceFile == "" {
		return false
	}
	var reason string
	path := kubeconfig.DisplayText(b.SourceFile, 400)
	switch k.Fate(b.SourceFile) {
	case FileGone:
		reason = "its kubeconfig file " + path + " is gone"
	case FileUnlisted:
		reason = "its kubeconfig source was removed (the file " + path + " is untouched)"
	default:
		return false
	}
	return s.disconnectBackend(b, "Disconnected from "+kubeconfig.DisplayText(b.Context, 200)+": "+reason+".")
}

// disconnectBackend closes b if it is still the connected backend and leaves the UI Disconnected,
// naming the reason. It reports whether it did. A connect that started meanwhile is not touched.
func (s *Session) disconnectBackend(b *Backend, reason string) bool {
	st := s.state
	st.mu.Lock()
	if st.closed || st.backend != b {
		st.mu.Unlock()
		return false
	}
	old := st.detachLocked()
	st.mu.Unlock()
	old.close(s.log)
	// A disconnect of Cogpit's own: an attempt that "failed", its line an error, so the context is
	// marked (connectlog.go).
	a := st.logs.begin(b.Context, AttemptDisconnect, LogError, reason)
	st.logs.finish(a, OutcomeFailed, LogError, "")
	s.UpdateUI(func(u *UIState) {
		// Connecting: b's own connect, between publishing b and saying Connected, which then
		// leaves the UI as this leaves it (connect_gone.go). A newer connect detaches b first.
		if u.Context != b.Context || !activeConnection(u.Connection.State) {
			return // another connect took over
		}
		u.Connection = Connection{State: Disconnected, Error: reason}
		u.View = ""
		u.Selected = nil
		u.Drawer = nil
		u.Filter = ""
	})
	s.emit(Event{Type: EventToast, Level: LevelWarning, Message: reason})
	return true
}

// closed reports whether the session was closed.
func (s *Session) closed() bool {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.closed
}
