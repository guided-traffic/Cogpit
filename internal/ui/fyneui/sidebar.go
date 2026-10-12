package fyneui

import (
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Tree node ids. A kind's node id is its kind key; a group's is groupPrefix and its path
// ("group:network/gatewayapi"); the API-group branches of Custom Resources are
// customGroupPrefix and the API group. A kind that a curated section places (Network › Gateway
// API › Gateways) is listed again under Custom Resources, as Lens does: that second node's id is
// aliasPrefix and the key, so ids stay unique, and selecting either opens the same view.
const (
	groupPrefix       = "group:"
	aliasPrefix       = "alias:"
	customGroup       = groupPrefix + "custom"
	customGroupPrefix = customGroup + "/"
	otherGroup        = groupPrefix + "other"
)

// Kind keys the shell refers to by name.
const (
	kindPods        = "pods"
	kindNamespaces  = "namespaces"
	kindEvents      = "events"
	kindDefinitions = "customresourcedefinitions"
)

// navEntry is one node of navLayout: a kind, a page, or a group of entries.
type navEntry struct {
	id    string     // a kind key, a page key, or groupPrefix + the group's path
	title string     // a group's title; for a kind, a title in place of its own (a CRD kind's plural title)
	items []navEntry // a group's entries, in Lens's order
	page  bool       // a page (engine.IsPage), shown while a cluster is connected
}

// page is a page of the navigator, titled as the engine titles it.
func page(key string) navEntry {
	title, _ := engine.PageTitle(key)
	return navEntry{id: key, title: title, page: true}
}

// navGroup builds a group; items are kind keys (strings) and navEntry values (titled kinds,
// sub-groups).
func navGroup(path, title string, items ...any) navEntry {
	g := navEntry{id: groupPrefix + path, title: title}
	for _, it := range items {
		switch e := it.(type) {
		case string:
			g.items = append(g.items, navEntry{id: e})
		case navEntry:
			g.items = append(g.items, e)
		}
	}
	return g
}

// titled is a kind the layout titles itself: CRD kinds in a curated section show Lens's plural
// title there ("Gateway Classes"), and their singular Kind name under Custom Resources.
func titled(key, title string) navEntry { return navEntry{id: key, title: title} }

// Gateway API kind keys (CRDs, docs/developer/architecture.md › Kind keys).
const (
	gatewayAPI  = ".gateway.networking.k8s.io"
	gatewayAPIX = ".gateway.networking.x-k8s.io"
)

// navLayout is the navigator's tree in the order of Lens's sidebar as captured from the running
// app (docs/parity/ui-parity.md, Cluster sidebar): top-level kinds and groups of kinds, with
// sub-groups (Config › Admission Policies, Network › Gateway API). Only kinds the cluster serves
// appear, and a group without any is left out: the Gateway API sub-group appears with its CRDs.
// Custom Resources is built from the cluster's kinds: Definitions, then one collapsed group per
// API group with its kinds by singular Kind name, the groups of curated sections included. Kinds
// the layout does not know and that no CRD defines go under Other. Pages (page) show while a
// cluster is connected: Workloads › Overview; the others that are not kinds (Overview,
// Applications, the Flux CD dashboard, Port Forwarding, Helm) come with their views. Listener Sets
// are listed under both groups Gateway API releases used for them.
var navLayout = []navEntry{
	{id: "nodes"},
	navGroup("workloads", "Workloads", page(engine.WorkloadsOverview),
		kindPods, "deployments", "daemonsets", "statefulsets", "replicasets", "replicationcontrollers", "jobs", "cronjobs"),
	navGroup("config", "Config",
		"configmaps", "secrets", "resourcequotas", "limitranges", "horizontalpodautoscalers",
		titled("verticalpodautoscalers.autoscaling.k8s.io", "Vertical Pod Autoscalers"),
		"poddisruptionbudgets", "priorityclasses", "runtimeclasses", "leases",
		"mutatingwebhookconfigurations", "validatingwebhookconfigurations",
		navGroup("config/admissionpolicies", "Admission Policies",
			"mutatingadmissionpolicies", "mutatingadmissionpolicybindings",
			"validatingadmissionpolicies", "validatingadmissionpolicybindings")),
	navGroup("network", "Network",
		"services", "endpointslices", "endpoints", "ingresses", "ingressclasses", "networkpolicies",
		navGroup("network/gatewayapi", "Gateway API",
			titled("gatewayclasses"+gatewayAPI, "Gateway Classes"),
			titled("gateways"+gatewayAPI, "Gateways"),
			titled("httproutes"+gatewayAPI, "HTTP Routes"),
			titled("backendtlspolicies"+gatewayAPI, "Backend TLS Policies"),
			titled("grpcroutes"+gatewayAPI, "gRPC Routes"),
			titled("xlistenersets"+gatewayAPIX, "Listener Sets"),
			titled("listenersets"+gatewayAPI, "Listener Sets"),
			titled("referencegrants"+gatewayAPI, "Reference Grants"),
			titled("tcproutes"+gatewayAPI, "TCP Routes"),
			titled("tlsroutes"+gatewayAPI, "TLS Routes"),
			titled("udproutes"+gatewayAPI, "UDP Routes")),
		kindPortForwards),
	navGroup("storage", "Storage", "persistentvolumeclaims", "persistentvolumes", "storageclasses"),
	navGroup("devices", "Device Resources", "resourceclaims", "resourceclaimtemplates", "deviceclasses", "resourceslices"),
	{id: kindNamespaces},
	{id: kindEvents},
	navGroup("helm", "Helm", "helmreleases"), // Charts come with chart repositories (G11)
	navGroup("access", "Access Control",
		"serviceaccounts", "clusterroles", "roles", "clusterrolebindings", "rolebindings", "podsecuritypolicies"),
	{id: customGroup, title: "Custom Resources", items: []navEntry{{id: kindDefinitions}}},
	{id: otherGroup, title: "Other"},
}

// Cogpit's own navigator icons, drawn for Cogpit (not taken from any other product): a network
// of three nodes, and stacked layers for namespaces.
var (
	iconNetwork = fyne.NewStaticResource("cogpit-network.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">`+
		`<rect x="9" y="2" width="6" height="6"/><rect x="2" y="16" width="6" height="6"/><rect x="16" y="16" width="6" height="6"/>`+
		`<rect x="11" y="8" width="2" height="4"/><rect x="4" y="11" width="16" height="2"/>`+
		`<rect x="4" y="11" width="2" height="5"/><rect x="18" y="11" width="2" height="5"/></svg>`))
	iconLayers = fyne.NewStaticResource("cogpit-layers.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24">`+
		`<path d="M12 2 22 7 12 12 2 7Z"/><path d="M4.2 10.9 12 14.8 19.8 10.9 22 12 12 17 2 12Z"/>`+
		`<path d="M4.2 15.9 12 19.8 19.8 15.9 22 17 12 22 2 17Z"/></svg>`))
)

// navIcons gives top-level items and groups an icon; kinds inside a group have none, as in Lens.
// The theme's icons are looked up when an item is drawn, never at package initialisation: no Fyne
// app runs then, and each lookup logged "Attempt to access current Fyne app when none is started"
// (ten lines at every start) and got the fallback theme's icon.
var navIcons = map[string]func() fyne.Resource{
	"nodes":                   theme.ComputerIcon,
	groupPrefix + "workloads": theme.GridIcon,
	groupPrefix + "config":    theme.SettingsIcon,
	groupPrefix + "network":   func() fyne.Resource { return iconNetwork },
	groupPrefix + "storage":   theme.StorageIcon,
	kindNamespaces:            func() fyne.Resource { return iconLayers },
	kindEvents:                theme.HistoryIcon,
	groupPrefix + "devices":   theme.ComputerIcon,
	groupPrefix + "helm":      theme.FileApplicationIcon,
	groupPrefix + "access":    theme.AccountIcon,
	groupPrefix + "custom":    theme.ListIcon,
	groupPrefix + "other":     theme.MoreHorizontalIcon,
}

// navIcon returns a navigator node's icon, or nil.
func navIcon(id string) fyne.Resource {
	if icon := navIcons[id]; icon != nil {
		return icon()
	}
	return nil
}

// sidebar is the navigator (navigator.go): the header with the scope dropdown and the search, and
// the tree whose root is the connected cluster with its views below. Opening a view dispatches
// ui.openView; the open view's row follows UIState.View.
type sidebar struct {
	sh       *shell
	head     *navHeader
	tree     *navTree
	children map[string][]string
	titles   map[string]string
	parents  map[string]string // node id → its branch's id; "" for a top-level item
	open     map[string]bool   // the groups the user opened (the window's, not engine state)
	revealed string            // the view whose branches selectView last opened
	marked   string            // the node of the open view
	filter   string            // the search text; "" shows every item
	shown    map[string]bool   // with a search: the nodes it keeps

	rootRow   *navRootRow
	badge     *clusterBadge
	name      *textLine
	subtitle  *textLine
	dot       *dot
	emptyHint *textLine
	hintRow   fyne.CanvasObject // the hint below the root while no item shows
	root      fyne.CanvasObject
}

func newSidebar(sh *shell) *sidebar {
	s := &sidebar{sh: sh, children: map[string][]string{}, titles: map[string]string{}, parents: map[string]string{},
		open: map[string]bool{}}
	s.badge = newClusterBadge()
	s.name = newTextLine("Not connected", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	s.subtitle = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	// The sidebar's width is the user's (panelwidths.go): a long context name is cut, not widened.
	s.name.truncate, s.subtitle.truncate = true, true
	s.dot = newDot(connectionColor(engine.Connection{}))
	s.emptyHint = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	s.emptyHint.truncate = true
	s.hintRow = inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, s.emptyHint)

	s.tree = newNavTree(s)
	s.head = newNavHeader(s)
	s.rootRow = newNavRootRow(s)
	// The header before the tree in the objects, so that Tab goes from the search field into the
	// tree (container.NewBorder would put the tree first).
	s.root = container.New(layout.NewBorderLayout(s.head.root, nil, nil, nil), s.head.root, s.tree)
	s.tree.rebuild()
	return s
}

// updateHeader shows the active context at the tree's root and in the scope dropdown: an initials
// badge, its name and what it is, and the connection's dot.
func (s *sidebar) updateHeader() {
	ui := s.sh.ui
	s.head.update()
	s.dot.SetColor(connectionColor(ui.Connection))
	if ui.Context == "" {
		s.badge.set("–", true)
		s.name.SetText("Not connected")
		s.subtitle.SetText("")
		return
	}
	name, file := splitContextID(ui.Context)
	s.badge.set(initials(name), false)
	s.name.SetText(name)
	switch {
	case ui.Connection.Demo:
		s.subtitle.SetText("Synthetic demo cluster")
	case ui.Connection.State != engine.Connected:
		s.subtitle.SetText(connectionText(ui.Connection))
	case file != "":
		s.subtitle.SetText(file) // the kubeconfig the context comes from (D-28: every context has one)
	default:
		s.subtitle.SetText("Kubernetes cluster")
	}
}

// setKinds rebuilds the tree from kinds.list. It runs on connect and whenever the cluster's
// custom resource kinds or the access reviews' answers change; branches keep the open or closed
// state they had, and new ones start closed (owner, 2026-10-12; Lens opened them, and the API
// groups of Custom Resources closed): the group of the open view is opened by selectView, so the
// selection stays visible. A kind the user may not list is left out; one not known yet is shown.
func (s *sidebar) setKinds(kinds []kindInfo) {
	kinds = listableKinds(kinds) // the View menu offers the same (menuaccess.go)
	s.children = map[string][]string{}
	s.titles = map[string]string{}
	s.parents = map[string]string{}

	have := make(map[string]kindInfo, len(kinds))
	for _, k := range kinds {
		have[k.Kind] = k
	}
	placed := map[string]bool{} // kinds the layout's sections show
	var place func(n navEntry)
	place = func(n navEntry) {
		if _, ok := have[n.id]; ok {
			placed[n.id] = true
		}
		for _, c := range n.items {
			place(c)
		}
	}
	for _, n := range navLayout {
		place(n)
	}

	var roots []string
	for _, n := range navLayout {
		if id := s.build(n, "", have, placed, kinds); id != "" {
			roots = append(roots, id)
		}
	}
	s.children[""] = roots
	for id := range s.open { // a branch that went away starts closed if it comes back
		if _, ok := s.children[id]; !ok {
			delete(s.open, id)
		}
	}
	s.refilter()
	s.tree.rebuild()
}

// build adds a layout entry the cluster serves to the tree and returns its node id, or "" when
// the cluster serves none of it.
func (s *sidebar) build(n navEntry, parent string, have map[string]kindInfo, placed map[string]bool, kinds []kindInfo) string {
	var kids []string
	switch {
	case n.id == customGroup:
		kids = s.buildCustom(have, placed, kinds)
	case n.id == otherGroup:
		for _, k := range kinds {
			if !placed[k.Kind] && !isCustomKind(k.Kind) {
				kids = append(kids, s.leaf(k.Kind, k.Title, n.id))
			}
		}
	case strings.HasPrefix(n.id, groupPrefix):
		for _, c := range n.items {
			if id := s.build(c, n.id, have, placed, kinds); id != "" {
				kids = append(kids, id)
			}
		}
	case n.page:
		if len(have) == 0 {
			return "" // not connected
		}
		return s.leaf(n.id, n.title, parent)
	default:
		k, ok := have[n.id]
		if !ok {
			return ""
		}
		title := n.title
		if title == "" {
			title = k.Title
		}
		return s.leaf(n.id, title, parent)
	}
	if len(kids) == 0 {
		return ""
	}
	s.children[n.id] = kids
	s.titles[n.id] = n.title
	s.parents[n.id] = parent
	return n.id
}

// buildCustom fills Custom Resources: Definitions, then one branch per API group, sorted by name,
// with its kinds by singular Kind name, sorted. A kind a curated section shows already gets an
// alias node here.
func (s *sidebar) buildCustom(have map[string]kindInfo, placed map[string]bool, kinds []kindInfo) []string {
	var kids []string
	if k, ok := have[kindDefinitions]; ok {
		kids = append(kids, s.leaf(kindDefinitions, k.Title, customGroup))
	}
	byGroup := map[string][]kindInfo{}
	for _, k := range kinds {
		if isCustomKind(k.Kind) {
			g := apiGroupOf(k)
			byGroup[g] = append(byGroup[g], k)
		}
	}
	groups := make([]string, 0, len(byGroup))
	for g := range byGroup {
		groups = append(groups, g)
	}
	slices.Sort(groups)
	for _, g := range groups {
		branch := customGroupPrefix + g
		members := byGroup[g]
		slices.SortFunc(members, func(a, b kindInfo) int {
			if c := strings.Compare(customTitle(a), customTitle(b)); c != 0 {
				return c
			}
			return strings.Compare(a.Kind, b.Kind)
		})
		var nodes []string
		for _, k := range members {
			id := k.Kind
			if placed[id] {
				id = aliasPrefix + id
			}
			nodes = append(nodes, s.leaf(id, customTitle(k), branch))
		}
		s.children[branch] = nodes
		s.titles[branch] = g
		s.parents[branch] = customGroup
		kids = append(kids, branch)
	}
	return kids
}

// leaf records a kind's node and returns its id.
func (s *sidebar) leaf(id, title, parent string) string {
	s.titles[id] = title
	s.parents[id] = parent
	return id
}

// isCustomKind reports whether a kind key names a kind from a custom resource definition: those
// keys are "<plural>.<group>", and built-in keys never contain a dot (architecture §4.5).
func isCustomKind(key string) bool { return strings.Contains(key, ".") }

// apiGroupOf is a custom kind's API group: kinds.list's group, else the key's.
func apiGroupOf(k kindInfo) string {
	if k.Group != "" {
		return k.Group
	}
	_, g, _ := strings.Cut(k.Kind, ".")
	return g
}

// customTitle is a kind's title under Custom Resources: its singular Kind name (captured).
func customTitle(k kindInfo) string {
	switch {
	case k.APIKind != "":
		return k.APIKind
	case k.Title != "":
		return k.Title
	}
	return k.Kind
}

// selectView marks the open view in the tree without dispatching anything: its node in a curated
// section if it has one, else its node under Custom Resources. When the view changes, the
// branches above it open, so that a view an agent opened inside a closed API group is visible, and
// the keyboard moves to it.
func (s *sidebar) selectView(view string) {
	if view == "" || s.titles[view] == "" {
		s.revealed, s.marked = "", ""
		s.tree.rebuild()
		return
	}
	changed := view != s.revealed
	if changed {
		s.revealed = view
		for p := s.parents[view]; p != ""; p = s.parents[p] {
			s.open[p] = true
		}
	}
	s.marked = view
	s.tree.rebuild()
	if changed {
		s.tree.setFocus(view)
	}
}

// activate does what a click or ↵ on a node does: open its view, or open or close a group (not
// while a search shows every group open); the root does nothing. An alias node opens its kind's
// view; the mark then moves to the kind's primary node.
func (s *sidebar) activate(id string) {
	switch {
	case id == "" || id == rootNode:
		return
	case s.isBranch(id):
		if s.filter == "" {
			s.setOpen(id, !s.open[id])
		}
		return
	}
	key := strings.TrimPrefix(id, aliasPrefix)
	if key != s.sh.ui.View {
		s.marked = id
		s.tree.rebuild()
		s.sh.dispatch("ui.openView", openViewParams{View: key}, func(_ any, err error) {
			if err != nil { // the view stays: so does its mark
				s.sh.showError(err)
				s.selectView(s.sh.ui.View)
			}
		})
	} else if key != id {
		s.selectView(key)
	}
}

// setOpen opens or closes a group.
func (s *sidebar) setOpen(id string, open bool) {
	if !s.isBranch(id) || s.open[id] == open {
		return
	}
	if open {
		s.open[id] = true
	} else {
		delete(s.open, id)
	}
	s.tree.rebuild()
}

// isBranch reports whether a node is a group.
func (s *sidebar) isBranch(id string) bool { return strings.HasPrefix(id, groupPrefix) }

// shownOpen reports whether a group shows its items: the user opened it, or a search is on.
func (s *sidebar) shownOpen(id string) bool { return s.filter != "" || s.open[id] }

// parentOf is the node above id in the tree: its group, or the root for a top-level item.
func (s *sidebar) parentOf(id string) string {
	if p := s.parents[id]; p != "" {
		return p
	}
	return rootNode
}

// depth is how many groups hold a node.
func (s *sidebar) depth(id string) int {
	d := 0
	for p := s.parents[id]; p != ""; p = s.parents[p] {
		d++
	}
	return d
}

// shownNodes lists the items the tree shows, in order, below the root: those of open groups, and
// with a search those it keeps.
func (s *sidebar) shownNodes() []string {
	var out []string
	var walk func(parent string)
	walk = func(parent string) {
		for _, id := range s.children[parent] {
			if s.shown != nil && !s.shown[id] {
				continue
			}
			out = append(out, id)
			if s.isBranch(id) && s.shownOpen(id) {
				walk(id)
			}
		}
	}
	walk("")
	return out
}

// setFilter applies the search text (the header's field).
func (s *sidebar) setFilter(text string) {
	text = strings.TrimSpace(text)
	if text == s.filter {
		return
	}
	s.filter = text
	s.refilter()
	s.tree.rebuild()
	if text != "" {
		s.tree.scroll.ScrollToTop() // the first matches
	} else {
		s.tree.reveal(s.tree.focus)
	}
}

// refilter works out which nodes the search keeps: an item whose title holds the text (ignoring
// case) with the groups above it, and a group whose title holds it with everything inside.
func (s *sidebar) refilter() {
	if s.filter == "" {
		s.shown = nil
		return
	}
	f := strings.ToLower(s.filter)
	s.shown = map[string]bool{}
	var visit func(id string, inMatch bool) bool
	visit = func(id string, inMatch bool) bool {
		match := inMatch || strings.Contains(strings.ToLower(s.titles[id]), f)
		keep := match
		for _, c := range s.children[id] {
			if visit(c, match) {
				keep = true
			}
		}
		if keep {
			s.shown[id] = true
		}
		return keep
	}
	for _, id := range s.children[""] {
		visit(id, false)
	}
}

// hint is the line below the root while no item shows: no cluster, or nothing matches the search.
func (s *sidebar) hint() string {
	switch {
	case len(s.children[""]) == 0:
		return "Connect a cluster to browse it."
	case s.filter != "" && len(s.shown) == 0:
		return "No items match “" + s.filter + "”."
	}
	return ""
}

// openFirstMatch opens the first view the search shows (↵ in the search field), and puts the
// keyboard there.
func (s *sidebar) openFirstMatch() {
	for _, id := range s.tree.visible {
		if id != rootNode && !s.isBranch(id) {
			s.tree.setFocus(id)
			s.activate(id)
			return
		}
	}
}

// initials returns up to two letters for a context's badge: "kind-cogpit" → "KC".
func initials(name string) string {
	parts := strings.FieldsFunc(name, func(r rune) bool { return r == '-' || r == '_' || r == '.' || r == '@' || r == ':' || r == '/' })
	if len(parts) == 0 {
		return "?"
	}
	var b []rune
	for _, p := range parts {
		r := []rune(p)
		if len(r) > 0 {
			b = append(b, r[0])
		}
		if len(b) == 2 {
			break
		}
	}
	if len(b) == 1 && len([]rune(parts[0])) > 1 {
		b = append(b, []rune(parts[0])[1])
	}
	return strings.ToUpper(string(b))
}
