package engine

import (
	"errors"
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// KubeconfigSources returns the connector's kubeconfig sources; nil when it has none.
func (c *ClusterConnector) KubeconfigSources() *KubeconfigSources { return c.Sources }

// withSourceContexts appends the sources' contexts to the default kubeconfig's. A source context
// id is resolved before the default files' names (connectOptions), so a default context that
// happens to carry such an id is left out, with a problem that says so.
func (c *ClusterConnector) withSourceContexts(out []ContextInfo, err error) ([]ContextInfo, error) {
	if c.Sources == nil {
		return out, err
	}
	extra, sErr := c.Sources.Contexts()
	problems := []error{err, sErr}
	ids := map[string]bool{}
	for _, x := range extra {
		ids[x.Name] = true
	}
	out = slices.DeleteFunc(out, func(ci ContextInfo) bool {
		if ids[ci.Name] {
			problems = append(problems, fmt.Errorf("the context %q of the default kubeconfig is hidden by the kubeconfig source context of the same id", ci.Name))
			return true
		}
		return false
	})
	return append(out, extra...), errors.Join(problems...)
}

// connectOptions returns the cluster options for a context id: a source context's file alone,
// with the sources' exec policy, and its id; or the default kubeconfig files and "" otherwise.
func (c *ClusterConnector) connectOptions(name string) (cluster.Options, string, error) {
	opts := cluster.Options{KubeconfigPaths: c.KubeconfigPaths, Context: name}
	if c.Sources == nil {
		return opts, "", nil
	}
	f, kctx, ok, err := c.Sources.Find(name)
	if err != nil {
		return opts, "", err
	}
	if !ok {
		return opts, "", nil
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
	// Fingerprint binds an approval to exactly this plugin (kubeconfig.approveExec takes it back).
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
// of the default kubeconfig files, or one without a plugin, is an error: no approval applies.
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
		return nil, fmt.Errorf("%q is not a context of a kubeconfig source added in Cogpit (the default kubeconfig's "+
			"credential plugins run as kubectl runs them)", id)
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

// ErrExecChangedSinceShown refuses an approval whose fingerprint is not the plugin's current one:
// the kubeconfig changed while the user looked at the dialog.
var ErrExecChangedSinceShown = errors.New("the credential plugin changed since it was shown; review it again")

// ApproveExec records the user's allowance of a source context's credential plugin, provided it
// is still the one the user was shown (fingerprint). Only the GUI's command calls it.
func (s *Session) ApproveExec(id, fingerprint string) (*ExecPluginInfo, error) {
	info, err := s.ExecPlugin(id)
	if err != nil {
		return nil, err
	}
	if info.Fingerprint != fingerprint {
		return nil, ErrExecChangedSinceShown
	}
	if err := s.KubeconfigSources().Approve(info.Plugin); err != nil {
		return nil, err
	}
	info.State = ExecApproved
	s.emit(Event{Type: EventContexts})
	return info, nil
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
				s.emit(Event{Type: EventContexts})
			}
		}
	}()
	return stop
}

// closed reports whether the session was closed.
func (s *Session) closed() bool {
	s.state.mu.Lock()
	defer s.state.mu.Unlock()
	return s.state.closed
}
