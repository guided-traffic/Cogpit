package engine

import (
	"fmt"

	"github.com/guided-traffic/cogpit/internal/kubeconfig"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Which cluster a command reads or acts on (D-69, several clusters connected at once). The engine
// asks it of the command's parameters, in this order:
//
//   - a context parameter (OnCluster, embedded in the parameters of a command that reads or acts on
//     a cluster without naming an object: resources.list, cluster.overview, terminal.open);
//   - the cluster of the dock tab the command works on (DockTargeted: a log, an editor, a shell);
//   - the context of the object reference in its "ref" parameter, or of its "refs";
//   - else the cluster the window shows.
//
// The target decides what the command's cluster is for Dispatch: whether a write is blocked while
// that cluster's connection is interrupted (writeBlocked), the access reviews it is checked against
// (permitted), the allowance of agent writes (D-2: per context) and the context the confirmation
// names. The command itself reads its backend through the same parameters (TargetBackend,
// resolve).

// OnCluster is the optional context parameter of a command that reads or acts on a connected
// cluster without naming an object (D-69 item 6): agents may work on any connected cluster, the
// shown one by default. Commands embed it in their parameters; a context that is not connected is
// an error, never a connect.
type OnCluster struct {
	Context string `json:"context,omitempty" jsonschema:"the connected cluster to use, by its context id as ui_state's connections list it; empty: the cluster the window shows. A context that is not connected is an error: cluster_connect it first"`
}

// TargetContext implements Targeted.
func (o OnCluster) TargetContext() string { return o.Context }

// Targeted is implemented by parameters that name the cluster a command uses (OnCluster).
type Targeted interface {
	TargetContext() string
}

// DockTargeted is implemented by the parameters of a command on one dock tab: the tab's cluster is
// the command's.
type DockTargeted interface {
	TargetDockTab() string
}

// targetContext is the cluster a command with parameters p reads or acts on, by the rules above;
// "" for the shown cluster.
func targetContext(s *Session, p any) string {
	if t, ok := p.(Targeted); ok {
		if c := t.TargetContext(); c != "" {
			return c
		}
	}
	if t, ok := p.(DockTargeted); ok {
		if tab, ok := s.UIState().DockTab(t.TargetDockTab()); ok && tab.Context != "" {
			return tab.Context
		}
	}
	for _, ref := range paramsRefs(p) {
		if ref.Context != "" {
			return ref.Context
		}
	}
	return ""
}

// oneCluster refuses objects that are in more than one cluster: one confirmation names one
// context (D-2), and each cluster's writes are its own.
func oneCluster(refs []model.ObjectRef) error {
	first := ""
	for _, r := range refs {
		switch {
		case r.Context == "":
		case first == "":
			first = r.Context
		case r.Context != first:
			return errSeveralClusters(first, r.Context)
		}
	}
	return nil
}

func errSeveralClusters(a, b string) error {
	return fmt.Errorf("the objects are in two clusters, %s and %s: act on one cluster at a time",
		kubeconfig.DisplayText(a, 200), kubeconfig.DisplayText(b, 200))
}
