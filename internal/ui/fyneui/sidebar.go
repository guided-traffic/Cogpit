package fyneui

import (
	"image/color"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

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
var navIcons = map[string]fyne.Resource{
	"nodes":                   theme.ComputerIcon(),
	groupPrefix + "workloads": theme.GridIcon(),
	groupPrefix + "config":    theme.SettingsIcon(),
	groupPrefix + "network":   iconNetwork,
	groupPrefix + "storage":   theme.StorageIcon(),
	kindNamespaces:            iconLayers,
	kindEvents:                theme.HistoryIcon(),
	groupPrefix + "devices":   theme.ComputerIcon(),
	groupPrefix + "helm":      theme.FileApplicationIcon(),
	groupPrefix + "access":    theme.AccountIcon(),
	groupPrefix + "custom":    theme.ListIcon(),
	groupPrefix + "other":     theme.MoreHorizontalIcon(),
}

// sidebar is the navigator: the connected cluster's identity on top, the tree of views below.
// Selecting a view dispatches ui.openView; the selection follows UIState.View.
type sidebar struct {
	sh        *shell
	tree      *widget.Tree
	children  map[string][]string
	titles    map[string]string
	parents   map[string]string // node id → its branch's id
	revealed  string            // the view whose branches selectView last opened
	marked    string            // the selected node; Fyne's tree does not report it
	selecting bool

	badge     *pill
	name      *textLine
	subtitle  *textLine
	emptyHint *textLine
	root      fyne.CanvasObject
}

func newSidebar(sh *shell) *sidebar {
	s := &sidebar{sh: sh, children: map[string][]string{}, titles: map[string]string{}, parents: map[string]string{}}
	s.tree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID { return s.children[uid] },
		// The root ("") is a branch too: Fyne's tree walks the root's children only if it is.
		func(uid widget.TreeNodeID) bool { return uid == "" || strings.HasPrefix(uid, groupPrefix) },
		func(bool) fyne.CanvasObject { return newNavItem() },
		func(uid widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			o.(*navItem).set(s.titles[uid], navIcons[uid], branch)
		},
	)
	s.tree.OnSelected = s.selected
	s.tree.HideSeparators = true

	s.badge = newPill("", colorAccent, true)
	s.badge.bold = true
	s.name = newTextLine("Not connected", theme.ColorNameForeground, theme.SizeNameText, fyne.TextStyle{Bold: true})
	s.subtitle = newTextLine("", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})
	s.emptyHint = newTextLine("Connect a cluster to browse it.", colorMuted, theme.SizeNameCaptionText, fyne.TextStyle{})

	header := inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding,
		container.NewBorder(nil, nil, container.NewCenter(s.badge), nil,
			container.NewVBox(s.name, s.subtitle)))
	s.root = container.NewBorder(
		container.NewVBox(header, newDivider(false)), nil, nil, nil,
		container.NewStack(s.tree, container.NewVBox(inset(theme.SizeNameInnerPadding, theme.SizeNameInnerPadding, s.emptyHint))))
	return s
}

// updateHeader shows the active context: an initials badge, its name and what it is.
func (s *sidebar) updateHeader() {
	ui := s.sh.ui
	if ui.Context == "" {
		s.badge.Set("–", colorMuted)
		s.name.SetText("Not connected")
		s.subtitle.SetText("")
		return
	}
	s.badge.Set(initials(ui.Context), colorAccent)
	s.name.SetText(ui.Context)
	switch {
	case ui.Connection.Demo:
		s.subtitle.SetText("Synthetic demo cluster")
	case ui.Connection.State == engine.Connected:
		s.subtitle.SetText("Kubernetes cluster")
	default:
		s.subtitle.SetText(connectionText(ui.Connection))
	}
}

// setKinds rebuilds the tree from kinds.list. It runs on connect and whenever the cluster's
// custom resource kinds or the access reviews' answers change; branches keep the open or closed
// state they had, new ones start open, except the API groups of Custom Resources, which start
// closed (captured). A kind the user may not list is left out; one not known yet is shown.
func (s *sidebar) setKinds(kinds []kindInfo) {
	kinds = slices.DeleteFunc(slices.Clone(kinds), func(k kindInfo) bool { return k.Listable != nil && !*k.Listable })
	wasOpen := map[string]bool{}
	for id := range s.children {
		if id != "" {
			wasOpen[id] = s.tree.IsBranchOpen(id)
		}
	}
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
	if len(roots) == 0 {
		s.emptyHint.Show()
	} else {
		s.emptyHint.Hide()
	}
	s.tree.Refresh()
	for id := range s.children {
		if id == "" {
			continue
		}
		open, known := wasOpen[id]
		if !known {
			open = !strings.HasPrefix(id, customGroupPrefix)
		}
		if open {
			s.tree.OpenBranch(id)
		} else {
			s.tree.CloseBranch(id)
		}
	}
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
// branches above it open, so that a view an agent opened inside a closed API group is visible.
func (s *sidebar) selectView(view string) {
	s.selecting = true
	defer func() { s.selecting = false }()
	if view == "" || s.titles[view] == "" {
		s.tree.UnselectAll()
		s.revealed, s.marked = "", ""
		return
	}
	if view != s.revealed {
		s.revealed = view
		for p := s.parents[view]; p != ""; p = s.parents[p] {
			s.tree.OpenBranch(p)
		}
	}
	s.tree.Select(view)
	s.marked = view
}

// selected opens a view, or toggles a group. An alias node opens its kind's view; the mark then
// moves to the kind's primary node.
func (s *sidebar) selected(uid widget.TreeNodeID) {
	if s.selecting {
		return
	}
	s.marked = uid
	if strings.HasPrefix(uid, groupPrefix) {
		s.tree.ToggleBranch(uid)
		s.selectView(s.sh.ui.View)
		return
	}
	key := strings.TrimPrefix(uid, aliasPrefix)
	if key != s.sh.ui.View {
		s.sh.dispatch("ui.openView", openViewParams{View: key}, nil)
	} else if key != uid {
		s.selectView(key)
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

// navItem is one row of the navigator: an icon for top-level entries and groups, and the title.
type navItem struct {
	widget.BaseWidget
	title  string
	icon   fyne.Resource
	branch bool

	img *iconImage
	txt *canvas.Text
}

func newNavItem() *navItem {
	n := &navItem{img: newIconImage(), txt: canvas.NewText("", color.Transparent)}
	n.ExtendBaseWidget(n)
	return n
}

func (n *navItem) set(title string, icon fyne.Resource, branch bool) {
	if title == n.title && icon == n.icon && branch == n.branch {
		return
	}
	n.title, n.icon, n.branch = title, icon, branch
	n.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (n *navItem) CreateRenderer() fyne.WidgetRenderer {
	r := &navItemRenderer{n: n}
	r.Refresh()
	return r
}

type navItemRenderer struct{ n *navItem }

func (r *navItemRenderer) Layout(size fyne.Size) {
	th := r.n.Theme()
	is := th.Size(theme.SizeNameInlineIcon)
	pad := th.Size(theme.SizeNamePadding)
	x := float32(0)
	if r.n.icon != nil {
		r.n.img.img.Move(fyne.NewPos(0, (size.Height-is)/2))
		r.n.img.img.Resize(fyne.NewSize(is, is))
		x = is + pad*2
	}
	ts := r.n.txt.MinSize()
	r.n.txt.Move(fyne.NewPos(x, (size.Height-ts.Height)/2))
	r.n.txt.Resize(ts)
}

func (r *navItemRenderer) MinSize() fyne.Size {
	th := r.n.Theme()
	ts := r.n.txt.MinSize()
	w := ts.Width
	if r.n.icon != nil {
		w += th.Size(theme.SizeNameInlineIcon) + th.Size(theme.SizeNamePadding)*2
	}
	return fyne.NewSize(w, ts.Height+2*th.Size(theme.SizeNameInnerPadding))
}

func (r *navItemRenderer) Refresh() {
	n := r.n
	th := n.Theme()
	n.txt.Text = n.title
	n.txt.TextSize = th.Size(theme.SizeNameText)
	n.txt.Color = themeColor(th, theme.ColorNameForeground)
	n.img.set(th, n.icon, colorMuted)
	r.Layout(n.Size())
	n.txt.Refresh()
}

func (r *navItemRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.n.img.img, r.n.txt}
}

func (r *navItemRenderer) Destroy() {}
