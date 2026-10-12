package engine

import (
	"errors"
	"fmt"
	"sort"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// A context's link (cluster.links, cluster.remove; the clusters' menus' "Remove from Cogpit…",
// ui-parity › Clusters). Cogpit reaches a context through Cogpit's link to a kubeconfig file or
// folder, a kubeconfig source (kubeconfigs.go); removing a cluster from Cogpit removes that link
// and never edits or deletes a kubeconfig file. Only a file the user added is unlinked that way,
// and with it every context the file defines. The rules (Cogpit's defaults, 2026-10-13, not the
// owner's decision; Lens's own rule is not captured):
//   - a context of a file in a sync folder, the default ~/.kube included, is refused: its link is
//     the folder's, and unlinking the folder would take every cluster in it (Kubeconfig Sources
//     removes a folder, after asking);
//   - a context of a file named by $KUBECONFIG or --kubeconfig is refused: the file is offered
//     again at the next start;
//   - the demo cluster has no kubeconfig to unlink.

// ContextLink is where Cogpit reads a context from, and whether cluster.remove can unlink it.
type ContextLink struct {
	// Context is the context id ("dev [dev.yaml]").
	Context string `json:"context"`
	// Source is the kubeconfig source the context comes from: an added file, a sync folder (the
	// default ~/.kube or one the user added) or a file named for this run.
	Source KubeconfigSource `json:"source"`
	// File is the kubeconfig file's path; Contexts are the ids of every context it defines, which
	// all leave Cogpit with the link.
	File     string   `json:"file"`
	Contexts []string `json:"contexts"`
	// Removable says whether cluster.remove can unlink it; Reason says why not.
	Removable bool   `json:"removable"`
	Reason    string `json:"reason,omitempty"`
}

// errNoSources refuses the link commands in a session without kubeconfig sources.
var errNoSources = errors.New("this Cogpit keeps no kubeconfig sources")

// ContextLinks returns the link of every context of the kubeconfig sources, sorted by context id.
// It reads the sources' files once (no cluster is asked).
func (s *Session) ContextLinks() ([]ContextLink, error) {
	k := s.KubeconfigSources()
	if k == nil {
		return nil, errNoSources
	}
	files, _, err := k.Files()
	if err != nil {
		return nil, err
	}
	out := []ContextLink{}
	for _, f := range files {
		if f.Err != nil {
			continue
		}
		ids := make([]string, 0, len(f.Contexts))
		for _, c := range f.Contexts {
			ids = append(ids, SourceContextID(c.Name, f.Label))
		}
		for _, id := range ids {
			out = append(out, linkOf(id, f, ids))
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Context < out[j].Context })
	return out, nil
}

// ContextLink returns one context's link. The demo cluster's says why it has none; a context no
// source offers is an error.
func (s *Session) ContextLink(id string) (ContextLink, error) {
	if id == DemoContext {
		return ContextLink{Context: id, Contexts: []string{id},
			Reason: "the demo cluster is Cogpit's own (--demo): it comes from no kubeconfig file"}, nil
	}
	k := s.KubeconfigSources()
	if k == nil {
		return ContextLink{}, errNoSources
	}
	f, _, ok, err := k.Find(id)
	if err != nil {
		return ContextLink{}, err
	}
	if !ok {
		return ContextLink{}, fmt.Errorf("%s is not a context of a kubeconfig source (cluster_contexts lists them)", kubeconfig.DisplayText(id, 200))
	}
	ids := make([]string, 0, len(f.Contexts))
	for _, c := range f.Contexts {
		ids = append(ids, SourceContextID(c.Name, f.Label))
	}
	return linkOf(id, f, ids), nil
}

// linkOf is a context's link through the file f, whose contexts are ids.
func linkOf(id string, f KubeconfigFile, ids []string) ContextLink {
	l := ContextLink{Context: id, Source: f.Source, File: f.Path, Contexts: ids, Removable: true}
	file := kubeconfig.DisplayText(f.Path, 400)
	folder := kubeconfig.DisplayText(f.Source.Path, 400)
	switch {
	case f.Source.Session:
		l.Removable, l.Reason = false, "named by $KUBECONFIG or --kubeconfig for this run: "+file+" is offered again at the "+
			"next start (take it out of $KUBECONFIG, or remove it for this run in Kubeconfig Sources)"
	case f.Source.Default:
		l.Removable, l.Reason = false, "in the synced folder "+folder+" (Cogpit's default): unlinking it would unlink the "+
			"whole folder and every cluster in it, which Kubeconfig Sources does after asking"
	case f.Source.Kind == SourceFolder:
		l.Removable, l.Reason = false, "in the synced folder "+folder+": unlinking it would unlink the whole folder and "+
			"every cluster in it, which Kubeconfig Sources does after asking"
	}
	return l
}

// CheckRemoveContext returns why cluster.remove cannot unlink a context, or nil.
func (s *Session) CheckRemoveContext(id string) error {
	l, err := s.ContextLink(id)
	if err != nil {
		return err
	}
	if !l.Removable {
		return errors.New(l.Reason)
	}
	return nil
}

// RemoveContext unlinks a context's kubeconfig file from Cogpit (cluster.remove): the file's
// source is removed as kubeconfig.remove removes it, so every context of the file leaves Cogpit,
// the credential plugins allowed for it are forgotten, and a connection made from it is closed.
// The file stays on disk as it is. It returns the link that was removed.
func (s *Session) RemoveContext(id string) (ContextLink, error) {
	l, err := s.ContextLink(id)
	if err != nil {
		return ContextLink{}, err
	}
	if !l.Removable {
		return l, errors.New(l.Reason)
	}
	src, err := s.KubeconfigSources().Remove(l.Source.ID)
	if src.ID != "" {
		s.CloseGoneSourceConnection() // a connection made from the file ends with the link
		s.emit(Event{Type: EventContexts})
	}
	if err != nil {
		return l, err
	}
	return l, nil
}
