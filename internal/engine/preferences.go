package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/guided-traffic/cogpit/internal/helm"
	"github.com/guided-traffic/cogpit/internal/kubectl"
)

// Preferences (G13, docs/parity/ui-parity.md › Preferences): Cogpit's own settings that are not a
// cluster's, as Lens's Preferences page holds them: the theme, the time zone absolute times are
// shown in, the tables' row height, the editor's font size, line numbers and tab size, and the
// local terminal's font size, shell and shell arguments, the proxy of the traffic that is no
// cluster's (the Helm repositories' downloads), and Helm's storage driver for new installs
// (HelmDriver, helmdriver.go). They are engine state (UIState.Preferences, the
// proxy's password redacted: Shown) so that agents see and change what the user sees; the GUI
// changes them only through preferences.set. They are kept in PreferencesFile under Cogpit's
// configuration directory, never in a kubeconfig, and no preference talks to a cluster. Only the
// user sets the proxy (ErrProxyGUIOnly, proxy.go).
//
// The local terminal's shell and its arguments name a program Cogpit runs on this computer when a
// terminal opens, so only the user may set them, in Cogpit's window (ErrShellGUIOnly): an agent
// that could set them and then open a terminal (terminal.open, a UI command) would run a program
// of its choice without the user's acceptance.

// PreferencesFile is the preferences' file under Cogpit's configuration directory.
const PreferencesFile = "preferences.json"

// Theme choices: Lens's "Sync with computer", Dark and Light.
const (
	ThemeSystem = "system"
	ThemeDark   = "dark"
	ThemeLight  = "light"
)

// Bounds of the preferences. Cogpit's defaults, inferred: Lens's own bounds are not captured.
const (
	DefaultFontSize    = 13 // the views' text size (the theme's), the editor's and the terminal's default
	MinFontSize        = 8
	MaxFontSize        = 32
	DefaultTabSize     = 2 // YAML indents by two spaces
	MaxTabSize         = 8
	MinTableRowHeight  = 20 // points; 0 keeps Cogpit's default (the text's height and 6 points above and below)
	MaxTableRowHeight  = 48
	maxShellPath       = 1024
	maxShellArgs       = 32
	maxShellArg        = 1024
	maxTimezoneName    = 64
	preferencesVersion = 1
)

// ErrNoPreferences disables the preferences commands of a session that keeps none (a headless one).
var ErrNoPreferences = errors.New("this Cogpit keeps no preferences")

// ErrShellGUIOnly refuses a change of the local terminal's shell or its arguments from anywhere but
// the user's own action in Cogpit's window.
var ErrShellGUIOnly = errors.New("only the user, in Cogpit's own window, can set the local terminal's shell and its " +
	"arguments; agents and the CLI are refused (they name a program Cogpit runs on this computer)")

// Preferences are Cogpit's own settings. The zero value of a field is not always its default:
// DefaultPreferences has them.
type Preferences struct {
	// Theme is ThemeSystem (Lens's "Sync with computer"), ThemeDark or ThemeLight. The app's
	// --theme flag overrides it for one run.
	Theme string `json:"theme"`
	// Timezone is the IANA time zone absolute times are shown in (the drawer's Created, the charts'
	// time axes and hover panels); empty: the computer's own.
	Timezone string `json:"timezone,omitempty"`
	// TableRowHeight is the height of a table row in points; 0: Cogpit's default.
	TableRowHeight int `json:"tableRowHeight,omitempty"`
	// EditorFontSize, EditorLineNumbers and EditorTabSize set the YAML editor of the Edit, Create
	// resource, Helm Upgrade and Helm Install tabs: its text size in points, whether it shows line
	// numbers, and how many spaces Tab indents by.
	EditorFontSize    int  `json:"editorFontSize"`
	EditorLineNumbers bool `json:"editorLineNumbers"`
	EditorTabSize     int  `json:"editorTabSize"`
	// TerminalFontSize is the local terminal's, pod shells' and attached sessions' text size.
	TerminalFontSize int `json:"terminalFontSize"`
	// TerminalShell is the local terminal's shell, an absolute path; empty: $SHELL, else /bin/zsh.
	// TerminalShellArgs are its arguments, one per entry. Pod shells are not affected.
	TerminalShell     string   `json:"terminalShell,omitempty"`
	TerminalShellArgs []string `json:"terminalShellArgs,omitempty"`
	// HelmDriver is Lens's HELM_DRIVER: where a new Helm install keeps its release's records,
	// "secret" (empty: Helm's default) or "configmap" (helmdriver.go). Releases already installed
	// keep the driver their records are in; "sql" is refused (helm.ErrSQLDriver).
	HelmDriver string `json:"helmDriver,omitempty"`
	// HTTPProxy is the proxy of the traffic that is no cluster's (Lens's "HTTP proxy for
	// non-cluster traffic"): the Helm repositories' indexes and charts and the kubectl downloads
	// (proxy.go, kubectl.go). Empty: the
	// environment's HTTPS_PROXY, as before. It may carry a password: the UI state and the
	// commands' results show it redacted, only the preferences file keeps it as typed.
	HTTPProxy string `json:"httpProxy,omitempty"`
	// TerminalRightClickCopyPaste: a right click in the dock's terminals copies the selection, or
	// pastes the clipboard when nothing is selected; off: it does neither (termpaste.go). Only the
	// user may set it (ErrRightClickGUIOnly).
	TerminalRightClickCopyPaste bool `json:"terminalRightClickCopyPaste,omitempty"`
	// KubectlDownload: a terminal whose kubectl does not suit its cluster offers to download the
	// matching release (D-3, kubectl.go); on by default, as Lens downloads by default.
	KubectlDownload bool `json:"kubectlDownload"`
	// KubectlMirror is the https base URL kubectl is downloaded from; empty: dl.k8s.io
	// (kubectl.DefaultMirror). KubectlDir is the directory for binaries, an absolute path; empty:
	// <user config dir>/Cogpit/kubectl. KubectlPath is the kubectl terminals run, an absolute
	// path; empty: the first on PATH. Only the user, in Cogpit's window, sets these three
	// (ErrKubectlGUIOnly).
	KubectlMirror string `json:"kubectlMirror,omitempty"`
	KubectlDir    string `json:"kubectlDir,omitempty"`
	KubectlPath   string `json:"kubectlPath,omitempty"`
}

// DefaultPreferences are the preferences of a first start: dark (Cogpit starts dark, as it did
// before the preference existed), the computer's time zone, the default row height, 13-point text,
// line numbers on, two-space tabs, the user's own shell, kubectl downloads offered.
func DefaultPreferences() Preferences {
	return Preferences{Theme: ThemeDark, EditorFontSize: DefaultFontSize, EditorLineNumbers: true,
		EditorTabSize: DefaultTabSize, TerminalFontSize: DefaultFontSize, KubectlDownload: true}
}

// Clone returns a deep copy.
func (p Preferences) Clone() Preferences {
	p.TerminalShellArgs = slices.Clone(p.TerminalShellArgs)
	return p
}

// Check says why preferences cannot be kept: the first field out of its bounds. The shell's file
// is not checked here (CheckShell).
func (p Preferences) Check() error {
	if err := CheckTheme(p.Theme); err != nil {
		return err
	}
	if _, err := LoadTimezone(p.Timezone); err != nil {
		return err
	}
	if p.TableRowHeight != 0 && (p.TableRowHeight < MinTableRowHeight || p.TableRowHeight > MaxTableRowHeight) {
		return fmt.Errorf("a table row is %d to %d points high (0: the default), not %d", MinTableRowHeight, MaxTableRowHeight, p.TableRowHeight)
	}
	for _, f := range []struct {
		what string
		size int
	}{{"the editor's", p.EditorFontSize}, {"the terminal's", p.TerminalFontSize}} {
		if f.size < MinFontSize || f.size > MaxFontSize {
			return fmt.Errorf("%s font size is %d to %d points, not %d", f.what, MinFontSize, MaxFontSize, f.size)
		}
	}
	if p.EditorTabSize < 1 || p.EditorTabSize > MaxTabSize {
		return fmt.Errorf("the editor's tab size is 1 to %d spaces, not %d", MaxTabSize, p.EditorTabSize)
	}
	if err := checkShellPath(p.TerminalShell); err != nil {
		return err
	}
	if err := checkShellArgs(p.TerminalShellArgs); err != nil {
		return err
	}
	if _, err := helm.ParseDriver(p.HelmDriver); err != nil {
		return err
	}
	if err := CheckProxyURL(p.HTTPProxy); err != nil {
		return err
	}
	return p.checkKubectl()
}

// checkKubectl refuses a download mirror that is not an https URL, and a directory for binaries
// or a path to kubectl that is not an absolute path of one line. The files are not checked here
// (CheckKubectlPath).
func (p Preferences) checkKubectl() error {
	if _, err := kubectl.CheckMirror(p.KubectlMirror); err != nil {
		return err
	}
	for _, f := range []struct{ what, path string }{{"the directory for binaries", p.KubectlDir}, {"the path to kubectl", p.KubectlPath}} {
		switch {
		case f.path == "":
		case len(f.path) > maxShellPath || !utf8.ValidString(f.path) || strings.IndexFunc(f.path, unicode.IsControl) >= 0:
			return fmt.Errorf("%s is one line of at most %d bytes", f.what, maxShellPath)
		case !filepath.IsAbs(f.path):
			return fmt.Errorf("%s %q: give an absolute path", f.what, f.path)
		}
	}
	return nil
}

// CheckKubectlPath says why a file cannot be the terminals' kubectl: not an absolute path, or no
// executable file there. Empty means the first kubectl on PATH.
func CheckKubectlPath(path string) error {
	if err := (Preferences{KubectlPath: path}).checkKubectl(); err != nil || path == "" {
		return err
	}
	fi, err := os.Stat(path)
	switch {
	case err != nil:
		return fmt.Errorf("the path to kubectl %s: no such file", path)
	case !fi.Mode().IsRegular():
		return fmt.Errorf("the path to kubectl %s is not a file", path)
	case runtime.GOOS != goosWindows && fi.Mode().Perm()&0o111 == 0:
		return fmt.Errorf("the path to kubectl %s is not executable", path)
	}
	return nil
}

// normalize replaces every field of preferences read from a file that Check would refuse by its
// default, so that one bad field does not cost the others.
func (p *Preferences) normalize() {
	d := DefaultPreferences()
	if CheckTheme(p.Theme) != nil {
		p.Theme = d.Theme
	}
	if _, err := LoadTimezone(p.Timezone); err != nil {
		p.Timezone = ""
	}
	if p.TableRowHeight != 0 && (p.TableRowHeight < MinTableRowHeight || p.TableRowHeight > MaxTableRowHeight) {
		p.TableRowHeight = 0
	}
	if p.EditorFontSize < MinFontSize || p.EditorFontSize > MaxFontSize {
		p.EditorFontSize = d.EditorFontSize
	}
	if p.TerminalFontSize < MinFontSize || p.TerminalFontSize > MaxFontSize {
		p.TerminalFontSize = d.TerminalFontSize
	}
	if p.EditorTabSize < 1 || p.EditorTabSize > MaxTabSize {
		p.EditorTabSize = d.EditorTabSize
	}
	if checkShellPath(p.TerminalShell) != nil {
		p.TerminalShell = ""
	}
	if checkShellArgs(p.TerminalShellArgs) != nil {
		p.TerminalShellArgs = nil
	}
	if len(p.TerminalShellArgs) == 0 {
		p.TerminalShellArgs = nil
	}
	if d, err := helm.ParseDriver(p.HelmDriver); err != nil || d == helm.DriverSecret {
		p.HelmDriver = "" // Helm's default
	} else {
		p.HelmDriver = string(d)
	}
	if p.HTTPProxy = strings.TrimSpace(p.HTTPProxy); CheckProxyURL(p.HTTPProxy) != nil {
		p.HTTPProxy = ""
	}
	if (Preferences{KubectlMirror: p.KubectlMirror}).checkKubectl() != nil {
		p.KubectlMirror = ""
	}
	if (Preferences{KubectlDir: p.KubectlDir}).checkKubectl() != nil {
		p.KubectlDir = ""
	}
	if (Preferences{KubectlPath: p.KubectlPath}).checkKubectl() != nil {
		p.KubectlPath = ""
	}
}

// CheckTheme refuses what is not a theme choice.
func CheckTheme(theme string) error {
	switch theme {
	case ThemeSystem, ThemeDark, ThemeLight:
		return nil
	}
	return fmt.Errorf("theme %q: system, dark or light", theme)
}

// LoadTimezone returns the time zone of a Timezone preference: the computer's own for "", else
// the IANA zone of that name ("Europe/Berlin", "UTC"). "Local" is refused: the empty name means it.
func LoadTimezone(name string) (*time.Location, error) {
	if name == "" {
		return time.Local, nil
	}
	if len(name) > maxTimezoneName || name == "Local" || strings.IndexFunc(name, notZoneRune) >= 0 ||
		strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return nil, fmt.Errorf("time zone %q: an IANA name such as Europe/Berlin or UTC, or empty for the computer's", name)
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("time zone %q is not in the zone database", name)
	}
	return loc, nil
}

func notZoneRune(r rune) bool {
	letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
	return !letter && !strings.ContainsRune("/_+-", r)
}

// checkShellPath refuses a shell that is not an absolute path of one line.
func checkShellPath(path string) error {
	switch {
	case path == "":
		return nil
	case len(path) > maxShellPath || !utf8.ValidString(path) || strings.IndexFunc(path, unicode.IsControl) >= 0:
		return fmt.Errorf("the shell is one line of at most %d bytes", maxShellPath)
	case !filepath.IsAbs(path):
		return fmt.Errorf("the shell %q: give its absolute path, such as /bin/zsh", path)
	}
	return nil
}

// CheckShell says why a shell cannot be the local terminal's: not an absolute path, or no
// executable file there. Empty means the default, $SHELL.
func CheckShell(path string) error {
	if err := checkShellPath(path); err != nil || path == "" {
		return err
	}
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("the shell %s: no such file", path)
	}
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("the shell %s is not a file", path)
	}
	if runtime.GOOS != goosWindows && fi.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("the shell %s is not executable", path)
	}
	return nil
}

// goosWindows is runtime.GOOS on Windows, where file modes say nothing about executing.
const goosWindows = "windows"

// checkShellArgs refuses arguments that are not lines of text: each argument is one line, as the
// Preferences page takes one per line.
func checkShellArgs(args []string) error {
	if len(args) > maxShellArgs {
		return fmt.Errorf("at most %d shell arguments", maxShellArgs)
	}
	for _, a := range args {
		if len(a) > maxShellArg || !utf8.ValidString(a) || strings.IndexFunc(a, unicode.IsControl) >= 0 {
			return fmt.Errorf("a shell argument is one line of at most %d bytes", maxShellArg)
		}
	}
	return nil
}

// PreferencesStore keeps the preferences in PreferencesFile under one directory (the app's:
// <user config dir>/Cogpit). Safe for concurrent use; as with the hotbar, two Cogpit processes
// keep one copy each and the last change wins.
type PreferencesStore struct {
	dir string // empty: kept in memory only

	// change serialises ChangePreferences from the write to the UI state it shows; mu guards the
	// state alone.
	change     sync.Mutex
	mu         sync.Mutex
	prefs      Preferences
	loc        *time.Location
	loaded     bool
	unreadable bool
}

// NewPreferencesStore returns the preferences kept in dir; an empty dir keeps them in memory only.
// Nothing is read or written until a session attaches it.
func NewPreferencesStore(dir string) *PreferencesStore { return &PreferencesStore{dir: dir} }

// Path returns the preferences' file; empty when they are kept in memory only.
func (p *PreferencesStore) Path() string {
	if p.dir == "" {
		return ""
	}
	return filepath.Join(p.dir, PreferencesFile)
}

type preferencesDoc struct {
	Version int `json:"version"`
	Preferences
}

// load reads the file once: a missing file gives the defaults; an unreadable one too, and is
// reported. p.mu is held.
func (p *PreferencesStore) load() error {
	if p.loaded {
		return nil
	}
	p.loaded = true
	p.prefs, p.loc = DefaultPreferences(), time.Local
	doc := preferencesDoc{Preferences: DefaultPreferences()}
	if err := readPrefsDoc(p.Path(), &doc); err != nil {
		p.unreadable = true
		return fmt.Errorf("the preferences file %s is unreadable (%v); Cogpit starts with the defaults and moves the file to %s.unreadable at the first change", p.Path(), err, PreferencesFile)
	}
	if doc.Version != 0 && doc.Version != preferencesVersion {
		p.unreadable = true
		return fmt.Errorf("the preferences file %s has format version %d, this Cogpit reads %d; it starts with the defaults and moves the file to %s.unreadable at the first change", p.Path(), doc.Version, preferencesVersion, PreferencesFile)
	}
	doc.normalize()
	p.prefs = doc.Preferences
	p.loc, _ = LoadTimezone(p.prefs.Timezone)
	return nil
}

// attach loads the preferences into the session's UI state; nil-safe.
func (p *PreferencesStore) attach(s *Session) {
	if p == nil {
		return
	}
	p.mu.Lock()
	err := p.load()
	st := p.prefs.Clone()
	p.mu.Unlock()
	if err != nil {
		s.log.Warn("preferences", "err", err)
	}
	st = st.Shown()
	s.ui.Preferences = &st
}

// Shown returns the preferences as they may be shown: the UI state, the commands' results. The
// proxy's password is redacted (proxy.go).
func (p Preferences) Shown() Preferences {
	p = p.Clone()
	p.HTTPProxy = RedactProxyURL(p.HTTPProxy)
	return p
}

// PreferencesStore returns the session's preferences; nil when it keeps none.
func (s *Session) PreferencesStore() *PreferencesStore { return s.opts.Preferences }

// Preferences returns the preferences as kept.
func (s *Session) Preferences() (Preferences, error) {
	p := s.opts.Preferences
	if p == nil {
		return DefaultPreferences(), ErrNoPreferences
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if err := p.load(); err != nil {
		s.log.Warn("preferences", "err", err)
	}
	return p.prefs.Clone(), nil
}

// ChangePreferences applies fn to a copy of the preferences, checks the result, writes it to the
// preferences' file and shows it (UIState.Preferences, EventUIState). If fn, the check or the
// write fails, nothing changes. Changes are serialised, so the UI states follow the writes.
func (s *Session) ChangePreferences(fn func(*Preferences) error) (Preferences, error) {
	p := s.opts.Preferences
	if p == nil {
		return DefaultPreferences(), ErrNoPreferences
	}
	p.change.Lock()
	defer p.change.Unlock()
	next, err := func() (Preferences, error) {
		p.mu.Lock()
		defer p.mu.Unlock()
		if err := p.load(); err != nil {
			s.log.Warn("preferences", "err", err)
		}
		next := p.prefs.Clone()
		if err := fn(&next); err != nil {
			return Preferences{}, err
		}
		if len(next.TerminalShellArgs) == 0 {
			next.TerminalShellArgs = nil
		}
		if next.HTTPProxy = strings.TrimSpace(next.HTTPProxy); next.HTTPProxy == RedactProxyURL(p.prefs.HTTPProxy) {
			next.HTTPProxy = p.prefs.HTTPProxy // the redacted form handed back as it was shown: unchanged
		}
		if err := next.Check(); err != nil {
			return Preferences{}, err
		}
		loc, _ := LoadTimezone(next.Timezone)
		if err := writePrefsDoc(p.Path(), &p.unreadable, preferencesDoc{Version: preferencesVersion, Preferences: next}); err != nil {
			return Preferences{}, fmt.Errorf("preferences: %w", err)
		}
		p.prefs, p.loc = next, loc
		return next.Clone(), nil
	}()
	if err != nil {
		return Preferences{}, err
	}
	shown := next.Shown() // the proxy without its password
	s.UpdateUI(func(u *UIState) { u.Preferences = &shown })
	return next, nil
}

// Location is the time zone absolute times are shown in: the Timezone preference's, else the
// computer's.
func (s *Session) Location() *time.Location {
	p := s.opts.Preferences
	if p == nil {
		return time.Local
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.loaded || p.loc == nil {
		return time.Local
	}
	return p.loc
}

// displayNow is the clock in the preferred time zone: the drawer's ages count from it and its
// absolute times are written in its zone (model.PropertiesSection).
func (s *Session) displayNow() time.Time { return s.Now().In(s.Location()) }

// The settings pages (Preferences, ⌘,; Cluster settings, ⌘⇧,) are UI state too, so that an agent
// sees which one the user has open and can open one for them.

// Settings pages and their sections, in the pages' order.
const (
	SettingsPreferences = "preferences"
	SettingsCluster     = "cluster"
)

// The settings pages' sections.
const (
	SectionApp        = "app"
	SectionKubernetes = "kubernetes"
	SectionEditor     = "editor"
	SectionTerminal   = "terminal"
	SectionTables     = "tables"
	SectionGeneral    = "general"
	SectionMetrics    = "metrics"
	SectionProxy      = "proxy"
	SectionNamespaces = "namespaces"
	SectionAgents     = "agents" // a cluster's agents: the allowance of their writes (allowance.go)
)

// PreferencesSections are the Preferences page's sections: App (theme, time zone), Proxy (the
// proxy of the traffic that is no cluster's), Kubernetes (kubeconfig syncs, kubectl for
// terminals), Editor, Terminal, Tables. Cogpit's grouping, inferred from Lens's page.
var PreferencesSections = []string{SectionApp, SectionProxy, SectionKubernetes, SectionEditor, SectionTerminal, SectionTables}

// ClusterSettingsSections are the Cluster settings page's built sections, in Lens's captured
// order (General, Proxy, Terminal, Namespaces, Metrics, ..., Node Shell) without the missing ones,
// then Cogpit's own Agents (allowance.go; its place inferred: last, Lens has no such section).
var ClusterSettingsSections = []string{SectionGeneral, SectionProxy, SectionTerminal, SectionNamespaces, SectionMetrics, SectionAgents}

// SettingsPage is the open settings page: Preferences, or one context's Cluster settings, and the
// section shown.
type SettingsPage struct {
	// Page is SettingsPreferences or SettingsCluster.
	Page string `json:"page"`
	// Context is the cluster page's context id, as cluster_contexts lists it.
	Context string `json:"context,omitempty"`
	// Section is the section shown, one of the page's (PreferencesSections, ClusterSettingsSections).
	Section string `json:"section"`
}

// CheckSection resolves a page's section: its first for "", else one of its own.
func CheckSection(page, section string) (string, error) {
	sections := PreferencesSections
	if page == SettingsCluster {
		sections = ClusterSettingsSections
	}
	if section == "" {
		return sections[0], nil
	}
	if !slices.Contains(sections, section) {
		return "", fmt.Errorf("section %q: one of %s", section, strings.Join(sections, ", "))
	}
	return section, nil
}
