package fyneui

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime/debug"
	"slices"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/engine/commands"
)

// welcomeEngine is a real engine over a kubeconfig file it never connects with (three contexts,
// kind-dev the current one) and the demo cluster, with a hotbar in a temporary folder.
func welcomeEngine(t *testing.T) *engine.Session {
	t.Helper()
	cfg := filepath.Join(t.TempDir(), "Cogpit")
	kc := filepath.Join(t.TempDir(), "config")
	if err := os.WriteFile(kc, []byte(settingsKubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	conn := &engine.ClusterConnector{KubeconfigPaths: []string{kc}, IncludeDemo: true}
	conn.DemoOptions.Pods, conn.DemoOptions.NoChurn, conn.DemoOptions.Seed = 200, true, 3
	s := engine.New(engine.Options{Connector: conn, Hotbar: engine.NewHotbar(cfg),
		Preferences: engine.NewPreferencesStore(cfg), ClusterSettings: engine.NewClusterSettingsStore(cfg)})
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s
}

// welcomeShown waits for the welcome page with its clusters listed.
func welcomeShown(t *testing.T, sh *shell, q *mainQueue) *welcomePage {
	t.Helper()
	w := sh.tabs.welcome
	q.until(t, "the welcome page", func() bool { return w.shown && w.root.Visible() && w.loaded && len(w.rows) > 0 })
	return w
}

// menuNamed returns the main menu's menu with this label, or nil.
func menuNamed(m *fyne.MainMenu, label string) *fyne.Menu {
	for _, menu := range m.Items {
		if menu.Label == label {
			return menu
		}
	}
	return nil
}

// Without a cluster the workspace shows the welcome page: the logo, the clusters the kubeconfigs
// offer (the current one first), Add Kubeconfigs and the documentation. A click on a cluster
// connects it through the engine and its tab replaces the page; Help › Welcome brings the page
// back over the connected cluster, its row back to the cluster's tab; Disconnect in the Cluster
// menu leaves the page with the cluster listed under this session.
func TestWelcomePageOverTheEngine(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a demo cluster")
	}
	s := welcomeEngine(t)
	sh, _, q := newEngineShell(t, s, Options{})
	w := welcomeShown(t, sh, q)
	if got := w.shownClusters(); len(got) != 4 || got[0] != "kind-dev" || !slices.Contains(got, demoLabel) {
		t.Fatalf("clusters on a first start %q, want kind-dev first and the demo", got)
	}
	if w.status.Visible() || !strings.Contains(w.hint.text, "finds a cluster by name") {
		t.Errorf("status %q (visible %v), hint %q", w.status.text, w.status.Visible(), w.hint.text)
	}

	// Pinned to the hotbar, the demo is listed under the hotbar, and only it.
	mustDispatch(t, s, "hotbar.addCluster", map[string]string{"context": engine.DemoContext})
	q.until(t, "the hotbar's group", func() bool { return slices.Equal(w.shownClusters(), []string{demoLabel}) })

	// One click connects through the engine; the cluster's tab takes the workspace.
	test.Tap(w.rows[0])
	q.until(t, "the demo connected", func() bool {
		return sh.ui.Connection.State == engine.Connected && sh.ui.Context == engine.DemoContext && len(sh.ui.Tabs) == 1
	})
	q.until(t, "the page gone", func() bool { return !w.shown && !w.root.Visible() })
	if u := s.UIState(); u.Welcome || u.ActiveTab == "" {
		t.Fatalf("connected: %+v", u)
	}
	// The connected cluster's menus offer Disconnect: the navigator's root and its hotbar tile; the
	// demo cluster has no Remove from Cogpit (it comes from no kubeconfig file).
	sh.side.rootRow.TappedSecondary(&fyne.PointEvent{})
	q.until(t, "the hotbar's tile", func() bool { return len(tilesOf(sh.hot)) == 1 })
	tilesOf(sh.hot)[0].TappedSecondary(&fyne.PointEvent{AbsolutePosition: fyne.NewPos(30, 100)})
	for name, m := range map[string]*fyne.Menu{"navigator root": sh.side.rootRow.menuModel, "hotbar tile": sh.hot.menuModel} {
		labels := menuLabels(m)
		if !slices.Contains(labels, "Disconnect") || slices.ContainsFunc(labels, func(l string) bool { return strings.HasPrefix(l, "Remove from Cogpit") }) {
			t.Errorf("%s menu %q", name, labels)
		}
	}

	// Help › Welcome over the connected cluster; its row goes back to its tab.
	help := menuNamed(sh.mainMenu(), "Help")
	if help == nil || help.Items[0].Label != "Welcome" {
		t.Fatalf("Help menu %v", menuLabels(help))
	}
	help.Items[0].Action()
	q.until(t, "the page over the cluster", func() bool { return w.shown && s.UIState().ActiveTab == "" })
	if !strings.HasPrefix(w.status.text, "Connected to demo") || sh.ui.Connection.State != engine.Connected {
		t.Errorf("status %q", w.status.text)
	}
	// ⌘W has no tab to close while the page shows: nothing happens, no error.
	toasts := len(sh.toasts.Objects)
	menuItemWithin(sh.mainMenu(), &desktop.CustomShortcut{KeyName: fyne.KeyW, Modifier: fyne.KeyModifierShortcutDefault}).Action()
	q.drain()
	if len(sh.toasts.Objects) != toasts || len(sh.ui.Tabs) != 1 {
		t.Errorf("⌘W on the welcome page: %d toasts (was %d), tabs %v", len(sh.toasts.Objects), toasts, sh.ui.Tabs)
	}
	q.until(t, "the connected row", func() bool { return len(w.rows) == 1 && w.rows[0].badges.Objects != nil })
	test.Tap(w.rows[0])
	q.until(t, "the tab back", func() bool { return !w.shown && sh.ui.ActiveTab != "" && sh.ui.View != "" })

	// Disconnect from the Cluster menu: the page, the demo under the hotbar.
	var disconnect *fyne.MenuItem
	for _, it := range sh.mainMenu().Items[0].Items {
		if it.Label == "Disconnect" {
			disconnect = it
		}
	}
	if disconnect == nil || disconnect.Disabled {
		t.Fatalf("Cluster menu %v", menuLabels(sh.mainMenu().Items[0]))
	}
	disconnect.Action()
	q.until(t, "disconnected", func() bool { return sh.ui.Connection.State == engine.Disconnected && w.shown })
	if s.Backend() != nil || len(sh.ui.Tabs) != 1 {
		t.Errorf("after the disconnect: backend %v, tabs %v", s.Backend(), sh.ui.Tabs)
	}
	q.until(t, "the menu follows", func() bool {
		for _, it := range sh.mainMenu().Items[0].Items {
			if it.Label == "Disconnect" {
				return it.Disabled
			}
		}
		return false
	})
}

// The welcome page's PNGs: a first start, no cluster connected.
func TestRenderWelcomeDark(t *testing.T)  { renderWelcome(t, ThemeDark) }
func TestRenderWelcomeLight(t *testing.T) { renderWelcome(t, ThemeLight) }

func renderWelcome(t *testing.T, mode ThemeMode) {
	s := welcomeEngine(t)
	sh, w, q := newEngineShell(t, s, Options{Theme: mode})
	welcomeShown(t, sh, q)
	q.drain()
	savePNG(t, w, "welcome_"+string(mode)+".png")
}

// sourcesWelcomeEngine is a real engine whose kubeconfig sources are a default folder (a fake
// ~/.kube with config) and an added file with two contexts, all in temporary folders.
func sourcesWelcomeEngine(t *testing.T) (s *engine.Session, added string) {
	t.Helper()
	root := t.TempDir()
	kube := filepath.Join(root, "home", ".kube")
	added = filepath.Join(root, "work", "team.yaml")
	for path, content := range map[string]string{
		filepath.Join(kube, "config"): settingsKubeconfig,
		added:                         strings.ReplaceAll(strings.ReplaceAll(settingsKubeconfig, "prod-eu-central", "team-a"), "staging", "team-b"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	k := engine.NewKubeconfigSources(filepath.Join(t.TempDir(), "Cogpit"), engine.WithDefaultFolder(kube))
	if _, _, err := k.AddFile(added); err != nil {
		t.Fatal(err)
	}
	s = engine.New(engine.Options{Connector: &engine.ClusterConnector{Sources: k}})
	if err := commands.RegisterAll(s.Registry()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	return s, added
}

// Remove from Cogpit in a context's menu: greyed with the reason for a context of the default
// ~/.kube; for an added file it asks first, naming the file's path and every context that leaves
// with it, and only then unlinks the file, which stays on disk unchanged.
func TestRemoveFromCogpitAsksFirst(t *testing.T) {
	s, added := sourcesWelcomeEngine(t)
	before, err := os.ReadFile(added)
	if err != nil {
		t.Fatal(err)
	}
	sh, win, q := newEngineShell(t, s, Options{})
	links := sh.tabs.welcome.links
	q.until(t, "the links", func() bool { return len(links.byCtx) == 6 })
	removeOf := func(id string) *fyne.MenuItem {
		for _, it := range sh.clusterItems(id) {
			if strings.HasPrefix(it.Label, "Remove from Cogpit") {
				return it
			}
		}
		return nil
	}
	if it := removeOf("kind-dev [.kube/config]"); it == nil || !it.Disabled || !strings.Contains(it.Label, "in the synced folder") {
		t.Fatalf("Remove of a ~/.kube context: %+v", it)
	}
	if it := removeOf(engine.DemoContext); it != nil {
		t.Errorf("Remove of the demo cluster: %+v", it)
	}
	if items := sh.clusterItems("team-a [team.yaml]"); len(items) != 2 || items[1].Label != "Remove from Cogpit…" || items[1].Disabled {
		t.Fatalf("an added file's context: %v", menuLabels(fyne.NewMenu("", items...)))
	}

	// The picker's row menu has it too.
	sh.picker.show()
	q.until(t, "the picker", func() bool { return len(sh.picker.contexts) == 6 })
	sh.picker.menu("team-a [team.yaml]", fyne.NewPos(300, 300))
	if labels := menuLabels(sh.picker.menuModel); !slices.Contains(labels, "Remove from Cogpit…") {
		t.Fatalf("picker menu %v", labels)
	}
	item := removeOf("team-a [team.yaml]")
	item.Action()
	q.until(t, "the confirmation", func() bool { return links.removing != nil })
	texts := strings.Join(dialogTexts(win), " ")
	for _, want := range []string{"Remove team-a from Cogpit?", added, "Only Cogpit's link to the file is removed", "team-a, team-b"} {
		if !strings.Contains(texts, want) {
			t.Errorf("the confirmation lacks %q: %s", want, texts)
		}
	}
	if sh.picker.visible() {
		t.Error("the picker lies over the confirmation")
	}
	ctxs, _ := s.Contexts()
	if len(ctxs) != 6 {
		t.Fatalf("removed before the confirmation: %d contexts", len(ctxs))
	}
	confirm := findButton(win, "Remove")
	if confirm == nil {
		t.Fatal("no Remove button")
	}
	test.Tap(confirm)
	q.until(t, "the file unlinked", func() bool { c, _ := s.Contexts(); return len(c) == 3 })
	ctxs, _ = s.Contexts()
	for _, c := range ctxs {
		if strings.HasSuffix(c.Name, "[team.yaml]") {
			t.Errorf("%s still offered", c.Name)
		}
	}
	if after, err := os.ReadFile(added); err != nil || !bytes.Equal(after, before) {
		t.Errorf("the kubeconfig file changed: %v", err)
	}
}

// The Help menu: Welcome, Documentation (Cogpit's README in the browser), About Cogpit, not labelled
// "About" (Fyne would move that into macOS's application menu); Copy MCP command only with an MCP
// server. Cogpit's Window menu exists on Windows and Linux only: macOS keeps GLFW's.
func TestHelpAndWindowMenus(t *testing.T) {
	browser := recordBrowser(t)
	f := newFakeEngine(t)
	reg := realCommands(t)
	for _, id := range []string{"ui.welcome.open", "cluster.disconnect"} {
		c, ok := reg.Get(id)
		if !ok {
			t.Fatalf("the registry has no %s", id)
		}
		probe[struct{}](t, f, c)
	}
	sh, _, q := newTestShell(t, f, Options{})
	help := menuNamed(sh.mainMenu(), "Help")
	if got := menuLabels(help); !slices.Equal(got, []string{"Welcome", "Documentation", "", "About Cogpit"}) {
		t.Errorf("Help menu %q", got)
	}
	help.Items[0].Action()
	q.until(t, "ui.welcome.open", func() bool { return len(f.callsOf("ui.welcome.open")) == 1 })
	help.Items[1].Action()
	if got := browser.opened(); !slices.Equal(got, []string{readmeURL}) {
		t.Errorf("Documentation opened %q", got)
	}
	sh.mcp = mcpState{on: true, command: "claude mcp add cogpit http://127.0.0.1:1/mcp"}
	sh.opts.MCPCommand = sh.mcp.command
	if got := menuLabels(menuNamed(sh.mainMenu(), "Help")); !slices.Contains(got, "Copy MCP command") {
		t.Errorf("Help menu with MCP %q", got)
	}

	if m := sh.windowMenu("darwin"); m != nil {
		t.Errorf("a Window menu on macOS: %q", menuLabels(m))
	}
	for _, goos := range []string{"linux", "windows"} {
		if got := menuLabels(sh.windowMenu(goos)); !slices.Equal(got, []string{"Toggle Full Screen", "", "Close Window"}) {
			t.Errorf("Window menu on %s: %q", goos, got)
		}
	}
	if sh.windowMenu("linux").Items[0].Action(); !sh.win.FullScreen() {
		t.Error("Toggle Full Screen")
	}
	// The Cluster menu: Disconnect greyed without a cluster.
	var disconnect *fyne.MenuItem
	for _, it := range sh.mainMenu().Items[0].Items {
		if it.Label == "Disconnect" {
			disconnect = it
		}
	}
	if disconnect == nil || !disconnect.Disabled {
		t.Errorf("Disconnect without a cluster: %+v", disconnect)
	}
}

// About Cogpit tells the packaged version, else the module's, the commit and the toolchain.
func TestReadAboutInfo(t *testing.T) {
	bi := &debug.BuildInfo{GoVersion: "go1.26.0", Main: debug.Module{Path: "github.com/guided-traffic/cogpit", Version: "(devel)"},
		Deps: []*debug.Module{{Path: "fyne.io/fyne/v2", Version: "v2.8.1"}},
		Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef0123"}, {Key: "vcs.time", Value: "2026-10-13T02:00:00Z"},
			{Key: "vcs.modified", Value: "true"}}}
	info := readAboutInfo("", bi)
	if info.Version != "development build" || info.Revision != "0123456789ab (modified)" || info.Go != "go1.26.0" || info.Fyne != "v2.8.1" {
		t.Errorf("a local build: %+v", info)
	}
	if got := strings.Join(info.lines(), "|"); got != "Commit: 0123456789ab (modified)|Committed: 2026-10-13T02:00:00Z|Go: go1.26.0|Fyne: v2.8.1|Licence: Apache License 2.0" {
		t.Errorf("lines %q", got)
	}
	bi.Main.Version = "v0.3.0"
	if info := readAboutInfo("", bi); info.Version != "v0.3.0" {
		t.Errorf("go install: %+v", info)
	}
	if info := readAboutInfo("0.4.0", bi); info.Version != "0.4.0" {
		t.Errorf("a packaged app: %+v", info)
	}
	if info := readAboutInfo("", nil); info.Version != "development build" || len(info.lines()) != 1 {
		t.Errorf("no build info: %+v", info)
	}
}
