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

const groupPrefix = "group:"

// Kind keys the shell refers to by name.
const (
	kindPods       = "pods"
	kindNamespaces = "namespaces"
)

// navLayout is the navigator's tree in the order of Lens's documented sidebar
// (docs/parity/ui-parity.md, Cluster sidebar): top-level kinds and groups of kinds. Only kinds the
// cluster serves appear, and a group without any is left out; kinds the layout does not know go
// under Other.
var navLayout = []struct {
	id    string   // a kind key for a top-level item, or groupPrefix + a group name
	title string   // the group's title
	kinds []string // the group's kind keys, in Lens's order
}{
	{id: "nodes"},
	{id: groupPrefix + "workloads", title: "Workloads", kinds: []string{
		kindPods, "deployments", "daemonsets", "statefulsets", "replicasets", "replicationcontrollers", "jobs", "cronjobs"}},
	{id: groupPrefix + "config", title: "Config", kinds: []string{
		"configmaps", "secrets", "resourcequotas", "limitranges", "horizontalpodautoscalers",
		"verticalpodautoscalers", "poddisruptionbudgets", "priorityclasses", "runtimeclasses", "leases",
		"mutatingwebhookconfigurations", "validatingwebhookconfigurations"}},
	{id: groupPrefix + "network", title: "Network", kinds: []string{
		"services", "endpoints", "endpointslices", "ingresses", "ingressclasses", "networkpolicies"}},
	{id: groupPrefix + "storage", title: "Storage", kinds: []string{
		"persistentvolumeclaims", "persistentvolumes", "storageclasses"}},
	{id: kindNamespaces},
	{id: "events"},
	{id: groupPrefix + "access", title: "Access Control", kinds: []string{
		"serviceaccounts", "clusterroles", "roles", "clusterrolebindings", "rolebindings"}},
	{id: groupPrefix + "other", title: "Other"},
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
	"events":                  theme.HistoryIcon(),
	groupPrefix + "access":    theme.AccountIcon(),
	groupPrefix + "other":     theme.MoreHorizontalIcon(),
}

// sidebar is the navigator: the connected cluster's identity on top, the tree of views below.
// Selecting a view dispatches ui.openView; the selection follows UIState.View.
type sidebar struct {
	sh        *shell
	tree      *widget.Tree
	children  map[string][]string
	titles    map[string]string
	selecting bool

	badge     *pill
	name      *textLine
	subtitle  *textLine
	emptyHint *textLine
	root      fyne.CanvasObject
}

func newSidebar(sh *shell) *sidebar {
	s := &sidebar{sh: sh, children: map[string][]string{}, titles: map[string]string{}}
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

// setKinds rebuilds the tree from kinds.list.
func (s *sidebar) setKinds(kinds []kindInfo) {
	s.children = map[string][]string{}
	s.titles = map[string]string{}
	have := map[string]bool{}
	for _, k := range kinds {
		have[k.Kind] = true
		s.titles[k.Kind] = k.Title
	}
	known := map[string]bool{}
	var roots []string
	for _, n := range navLayout {
		if !strings.HasPrefix(n.id, groupPrefix) {
			known[n.id] = true
			if have[n.id] {
				roots = append(roots, n.id)
			}
			continue
		}
		var kids []string
		for _, k := range n.kinds {
			known[k] = true
			if have[k] {
				kids = append(kids, k)
			}
		}
		if n.id == groupPrefix+"other" {
			for _, k := range kinds {
				if !known[k.Kind] && !slices.Contains(kids, k.Kind) {
					kids = append(kids, k.Kind)
				}
			}
		}
		if len(kids) > 0 {
			s.children[n.id] = kids
			s.titles[n.id] = n.title
			roots = append(roots, n.id)
		}
	}
	s.children[""] = roots
	if len(roots) == 0 {
		s.emptyHint.Show()
	} else {
		s.emptyHint.Hide()
	}
	s.tree.Refresh()
	s.tree.OpenAllBranches()
}

// selectView marks the open view in the tree without dispatching anything.
func (s *sidebar) selectView(view string) {
	s.selecting = true
	defer func() { s.selecting = false }()
	if view == "" || s.titles[view] == "" {
		s.tree.UnselectAll()
		return
	}
	s.tree.Select(view)
}

// selected opens a view, or toggles a group.
func (s *sidebar) selected(uid widget.TreeNodeID) {
	if s.selecting {
		return
	}
	if strings.HasPrefix(uid, groupPrefix) {
		s.tree.ToggleBranch(uid)
		s.selectView(s.sh.ui.View)
		return
	}
	if uid != s.sh.ui.View {
		s.sh.dispatch("ui.openView", openViewParams{View: uid}, nil)
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
