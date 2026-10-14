// Package commands registers Cogpit's engine commands, one file per area (ui.go, resources.go,
// dock.go, ...). Every command becomes a menu item, a palette entry and an MCP tool through the
// registry; the GUI and agents dispatch the same commands.
//
// Layer L3, with package engine.
package commands

import (
	"errors"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Kind keys the commands refer to.
const (
	kindPods        = "pods"
	kindDeployments = "deployments"
)

// RegisterAll registers every command. It fails on the first registration error (a duplicate id
// is a programming error found at start-up).
func RegisterAll(r *engine.Registry) error {
	g := &registrar{r: r}
	registerUI(g)
	registerCluster(g)
	registerResources(g)
	registerDock(g)
	registerDockOutput(g)
	registerActions(g)
	registerSelection(g)
	registerNodes(g)
	registerNodeShell(g)
	registerWorkloads(g)
	registerEdit(g)
	registerDrafts(g)
	registerRevert(g)
	registerAccess(g)
	registerPortForward(g)
	registerSecrets(g)
	registerHelm(g)
	registerHelmUpgrade(g)
	registerHelmRepos(g)
	registerHelmHTTP(g)
	registerHelmInstall(g)
	registerChartDrawer(g)
	registerWorkloadActions(g)
	registerAttach(g)
	registerLogs(g)
	registerCopy(g)
	registerTables(g)
	registerViewRows(g)
	registerStatusFilter(g)
	registerOverview(g)
	registerClusterOverview(g)
	registerKubeconfigs(g)
	registerMetrics(g)
	registerMetricsRange(g)
	registerDrawerTables(g)
	registerHotbar(g)
	registerCreateDialogs(g)
	registerSecretCreate(g)
	registerPalette(g)
	registerApplications(g)
	registerArgoCD(g)
	registerPreferences(g)
	registerClusterSettings(g)
	registerAllowance(g)
	registerAudit(g)
	registerDiagnostics(g)
	registerTabs(g)
	registerWelcome(g)
	registerClusterMenu(g)
	registerNotifications(g)
	registerNavScope(g)
	registerKubectl(g)
	registerFlux(g)
	return g.err
}

// registrar registers commands and keeps the first error.
type registrar struct {
	r   *engine.Registry
	err error
}

func add[P any](g *registrar, spec engine.Spec[P]) {
	if g.err == nil {
		g.err = engine.Register(g.r, spec)
	}
}

// labelDelete is the label of every kind's last action, as Lens labels it: object.delete, and the
// commands that take its place for kinds that are not API objects (Helm releases, port forwards).
const labelDelete = "Delete"

// cacheReads are the verbs of commands that read through the cache: the informer lists and
// watches the kind they read. The kind is a parameter, hence the wildcard.
var cacheReads = []engine.Verb{{Verb: "list", Resource: "*"}, {Verb: "watch", Resource: "*"}}

// connected is the Enabled check of commands that need a cluster.
func connected(s *engine.Session) (*engine.Backend, error) {
	b := s.Backend()
	if b == nil {
		return nil, errors.New("no cluster connected")
	}
	return b, nil
}

// writable is the Enabled check of commands that change a cluster.
func writable(s *engine.Session) (*engine.Backend, error) {
	b, err := connected(s)
	if err != nil {
		return nil, err
	}
	if !b.CanWrite() {
		return nil, errors.New("the demo cluster cannot be changed")
	}
	return b, nil
}

// streamable is the Enabled check of commands that open logs, shells or terminals.
func streamable(s *engine.Session, what string) (*engine.Backend, error) {
	b, err := connected(s)
	if err != nil {
		return nil, err
	}
	if b.Conn == nil {
		return nil, errors.New(what + " are not available on the demo cluster")
	}
	return b, nil
}

// cachedTarget is the Enabled check of actions on one object: the reference must be of the kind
// and the object must be in the cache.
func cachedTarget(s *engine.Session, ref model.ObjectRef, kind, kindErr string) error {
	if ref.Kind != kind {
		return errors.New(kindErr)
	}
	if _, err := s.Lookup(ref); err != nil {
		return err
	}
	return nil
}

// targetOf is the Targets function of commands on one object: the resolved reference, so that
// the confirmation names its namespace and context.
func targetOf(s *engine.Session, ref model.ObjectRef) ([]model.ObjectRef, error) {
	r, _, err := s.ResolveRef(ref)
	if err != nil {
		return nil, err
	}
	return []model.ObjectRef{r}, nil
}
