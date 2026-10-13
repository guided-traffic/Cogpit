package commands

import (
	"context"
	"slices"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Preferences (G13; engine/preferences.go): Cogpit's own settings, the Preferences page (⌘,).
// preferences.set changes only Cogpit's preferences file, never a cluster or a kubeconfig, so it
// is a UI command that agents run freely (D-2), except the local terminal's shell and its
// arguments, which name a program Cogpit runs and are refused for every origin but the GUI
// (engine.ErrShellGUIOnly). The open page is UI state (settings.close closes it).

// PreferencesResult is what preferences.get and preferences.set return.
type PreferencesResult struct {
	Preferences engine.Preferences `json:"preferences"`
	// Timezone is the zone absolute times are shown in now: the preference's, or "Local" for the
	// computer's own.
	Timezone string `json:"timezone"`
	// Kept says whether the preferences outlive this run (false without a configuration folder).
	Kept bool `json:"kept"`
}

type preferencesSetParams struct {
	Theme             *string  `json:"theme,omitempty" jsonschema:"system (follow the computer's light or dark setting), dark or light; Cogpit's window switches at once unless it was started with --theme"`
	Timezone          *string  `json:"timezone,omitempty" jsonschema:"IANA time zone absolute times are shown in, such as Europe/Berlin or UTC; empty: the computer's own"`
	TableRowHeight    *int     `json:"tableRowHeight,omitempty" jsonschema:"a table row's height in points, 20 to 48; 0: Cogpit's default"`
	EditorFontSize    *int     `json:"editorFontSize,omitempty" jsonschema:"the YAML editor's text size in points, 8 to 32 (default 13)"`
	EditorLineNumbers *bool    `json:"editorLineNumbers,omitempty" jsonschema:"whether the YAML editor shows line numbers (default true)"`
	EditorTabSize     *int     `json:"editorTabSize,omitempty" jsonschema:"how many spaces Tab indents by in the YAML editor, 1 to 8 (default 2)"`
	TerminalFontSize  *int     `json:"terminalFontSize,omitempty" jsonschema:"the dock terminals' text size in points, 8 to 32 (default 13), for terminals opened afterwards"`
	TerminalShell     *string  `json:"terminalShell,omitempty" jsonschema:"the local terminal's shell, an absolute path; empty: $SHELL. Only the user in Cogpit's window may set it: refused for agents"`
	TerminalShellArgs []string `json:"terminalShellArgs,omitempty" jsonschema:"the local terminal shell's arguments, one per entry. Only the user in Cogpit's window may set them: refused for agents"`
	HTTPProxy         *string  `json:"httpProxy,omitempty" jsonschema:"the proxy of the traffic that is no cluster's (the Helm repositories' indexes and charts): http://<address>:<port>, https:// or socks5://, optionally with user:password@; empty: the environment's HTTPS_PROXY. Agents and the CLI may only clear it (an empty string): setting one is the user's, in Cogpit's window"`
}

// shellChange reports whether the parameters change the local terminal's shell or arguments.
func (p preferencesSetParams) shellChange() bool {
	return p.TerminalShell != nil || p.TerminalShellArgs != nil
}

type preferencesOpenParams struct {
	Section string `json:"section,omitempty" jsonschema:"the section to show: app (theme, time zone), proxy, kubernetes, editor, terminal or tables; omitted: app"`
}

// prefsResult is what the preferences commands return: the proxy's password as ••• (Shown).
func prefsResult(s *engine.Session, p engine.Preferences) PreferencesResult {
	return PreferencesResult{Preferences: p.Shown(), Timezone: s.Location().String(), Kept: s.PreferencesStore().Path() != ""}
}

const prefsWritesOnly = " Changes only Cogpit's own preferences (a file in Cogpit's configuration directory), never a cluster or a kubeconfig."

func registerPreferences(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "preferences.get", Label: "Show Preferences", Danger: engine.DangerRead,
		Description: "Returns Cogpit's preferences as the Preferences page (⌘,) shows them: the theme (system, dark, light), " +
			"the time zone absolute times are shown in (empty: the computer's) and the zone in effect, the tables' row " +
			"height, the YAML editor's font size, line numbers and tab size, the terminal's font size, and the local " +
			"terminal's shell and arguments (empty: $SHELL), the HTTP proxy of the traffic that is no cluster's (empty: the " +
			"environment's; its password shown as •••); kept says whether they outlive this run. ui_state carries the same.",
		Enabled: func(s *engine.Session, _ struct{}) error { _, err := s.Preferences(); return err },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			p, err := s.Preferences()
			if err != nil {
				return nil, err
			}
			return prefsResult(s, p), nil
		},
	})

	add(g, engine.Spec[preferencesSetParams]{
		ID: "preferences.set", Label: "Set Preferences", Danger: engine.DangerUI,
		Description: "Changes some of Cogpit's preferences, as the Preferences page does; the fields left out stay as they are. " +
			"The window follows at once: the theme, the time zone of the drawer's Created and the charts, the tables' row " +
			"height, the open editors; the HTTP proxy at the next download from a Helm repository. The local terminal's " +
			"shell and its arguments are refused for agents and the CLI: they name a program Cogpit runs on this " +
			"computer; so is setting the HTTP proxy (it decides where Cogpit's downloads go), which they may only clear." +
			prefsWritesOnly + " Returns the preferences.",
		Enabled: func(s *engine.Session, _ preferencesSetParams) error { _, err := s.Preferences(); return err },
		Refuse: func(ctx context.Context, _ *engine.Session, p preferencesSetParams) error {
			if p.shellChange() && engine.OriginOf(ctx) != engine.OriginGUI {
				return engine.ErrShellGUIOnly
			}
			return refuseProxy(ctx, p.HTTPProxy)
		},
		Run: func(ctx context.Context, s *engine.Session, p preferencesSetParams) (any, error) {
			if p.shellChange() && engine.OriginOf(ctx) != engine.OriginGUI {
				return nil, engine.ErrShellGUIOnly
			}
			if err := refuseProxy(ctx, p.HTTPProxy); err != nil {
				return nil, err
			}
			if p.TerminalShell != nil {
				if err := engine.CheckShell(strings.TrimSpace(*p.TerminalShell)); err != nil {
					return nil, err
				}
			}
			next, err := s.ChangePreferences(func(pr *engine.Preferences) error {
				setIf(&pr.Theme, p.Theme)
				if p.Timezone != nil {
					pr.Timezone = strings.TrimSpace(*p.Timezone)
				}
				setIf(&pr.TableRowHeight, p.TableRowHeight)
				setIf(&pr.EditorFontSize, p.EditorFontSize)
				setIf(&pr.EditorLineNumbers, p.EditorLineNumbers)
				setIf(&pr.EditorTabSize, p.EditorTabSize)
				setIf(&pr.TerminalFontSize, p.TerminalFontSize)
				if p.TerminalShell != nil {
					pr.TerminalShell = strings.TrimSpace(*p.TerminalShell)
				}
				if p.TerminalShellArgs != nil {
					pr.TerminalShellArgs = slices.Clone(p.TerminalShellArgs)
				}
				if p.HTTPProxy != nil {
					next, err := engine.MergeProxyPassword(pr.HTTPProxy, *p.HTTPProxy) // "•••" keeps the password set before
					if err != nil {
						return err
					}
					pr.HTTPProxy = next
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
			return prefsResult(s, next), nil
		},
	})

	add(g, engine.Spec[preferencesOpenParams]{
		ID: "preferences.open", Label: "Preferences", MenuPath: "Cogpit/Preferences", Shortcut: "Cmd+,", Danger: engine.DangerUI,
		Description: "Opens Cogpit's Preferences page in the window, as ⌘, (Ctrl+,) does, at a section: app (theme, time zone), " +
			"proxy (the HTTP proxy of the traffic that is no cluster's), kubernetes (kubeconfig syncs), editor, terminal or " +
			"tables. It takes the place of the cluster's views until " +
			"closed (settings_close, Escape); ui_state reports it as settings. Returns the new UI state.",
		Enabled: func(s *engine.Session, p preferencesOpenParams) error {
			if _, err := s.Preferences(); err != nil {
				return err
			}
			_, err := engine.CheckSection(engine.SettingsPreferences, p.Section)
			return err
		},
		Run: func(_ context.Context, s *engine.Session, p preferencesOpenParams) (any, error) {
			section, err := engine.CheckSection(engine.SettingsPreferences, p.Section)
			if err != nil {
				return nil, err
			}
			return s.UpdateUI(func(u *engine.UIState) {
				u.Settings = &engine.SettingsPage{Page: engine.SettingsPreferences, Section: section}
				u.Palette = false
			}), nil
		},
	})

	add(g, engine.Spec[struct{}]{
		ID: "settings.close", Label: "Close Settings", Danger: engine.DangerUI,
		Description: "Closes the open settings page, Preferences or a cluster's Settings, as its close button or Escape does; " +
			"the cluster's views come back as they were. Returns the new UI state.",
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			return s.UpdateUI(func(u *engine.UIState) { u.Settings = nil }), nil
		},
	})
}

// setIf sets *dst to *v when v is given.
func setIf[T any](dst *T, v *T) {
	if v != nil {
		*dst = *v
	}
}
