package commands

import (
	"context"
	"slices"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Cluster settings (G13; engine/clustersettings.go): what the user sets for one context in Cogpit,
// the Cluster Settings page (⌘⇧,): its display name and icon (a colour of the theme's accent
// palette, or an uploaded image), and its terminals' working directory and kubectl namespace.
// Every command changes only Cogpit's own cluster settings file (and its icons folder), never a
// cluster or a kubeconfig, so they are UI commands that agents run freely (D-2), except the icon
// upload, which reads a file on this computer and is the user's own action in the window
// (engine.ErrIconGUIOnly). The Prometheus a cluster's metrics come from stays cluster.setPrometheus.

type clusterSettingsParams struct {
	Context string `json:"context,omitempty" jsonschema:"context id as cluster_contexts lists it (a bare context name works when one file only defines it); empty or omitted: the connected context"`
}

type clusterSettingsSetParams struct {
	Context              string   `json:"context,omitempty" jsonschema:"context id as cluster_contexts lists it (a bare context name works when one file only defines it); empty or omitted: the connected context"`
	Name                 *string  `json:"name,omitempty" jsonschema:"the cluster's display name in the hotbar, the context picker and the navigator, one line of at most 128 characters; empty: the context's name"`
	IconColor            *string  `json:"iconColor,omitempty" jsonschema:"the icon's colour, one of cyan, blue, violet, magenta, red, orange, yellow, green (the theme's accents); it replaces an uploaded image; empty: the accent"`
	TerminalDir          *string  `json:"terminalDir,omitempty" jsonschema:"the working directory of the cluster's local terminals: an absolute path, or one starting with $HOME or ~; empty: $HOME"`
	TerminalNamespace    *string  `json:"terminalNamespace,omitempty" jsonschema:"kubectl's namespace in the cluster's local terminals; empty: the namespace filter's single namespace, else the context's own"`
	Proxy                *string  `json:"proxy,omitempty" jsonschema:"the HTTP(S) or SOCKS5 proxy of this cluster's API traffic (http://<address>:<port>, https:// or socks5://, optionally with user:password@), applied at the next connect in place of the kubeconfig's proxy-url; empty: none. Agents and the CLI may only clear it (an empty string): setting one is the user's, in Cogpit's window"`
	HiddenMetrics        []string `json:"hiddenMetrics,omitempty" jsonschema:"the metrics the window hides for this cluster, and does not ask the cluster for: places (cluster: the Overview's charts; nodes, pods: their lists' CPU and Memory columns and their drawers' charts; containers: a pod's containers' charts) and measures (cpu, memory, network, filesystem), or all; it replaces the list set before, and an empty list shows every metric again. metrics_get still answers"`
	MetricsSource        *string  `json:"metricsSource,omitempty" jsonschema:"which providers the cluster's metrics come from, as Lens's Metrics source: auto (both: Prometheus's history and the Metrics Server's current values, the default), prometheus (Prometheus alone), metrics-server (the Metrics Server alone: no history, network, filesystem or disk) or none (no metrics: nothing is asked); applied to the connected cluster at once"`
	AccessibleNamespaces []string `json:"accessibleNamespaces,omitempty" jsonschema:"the namespaces the user may read in, for a user who may not list the cluster's namespaces (Lens: Accessible namespaces), at most 100 namespace names; it replaces the list set before, and an empty list removes them. It only narrows what Cogpit reads: the namespace filter offers them, and a kind the user may not list in all namespaces is watched in each of them instead"`
}

type clusterIconParams struct {
	Context string `json:"context,omitempty" jsonschema:"context id as cluster_contexts lists it; empty or omitted: the connected context"`
	Path    string `json:"path" jsonschema:"the image file the user picked: .png, .jpeg or .svg, at most 1 MiB"`
}

type clusterSettingsOpenParams struct {
	Context string `json:"context,omitempty" jsonschema:"context id as cluster_contexts lists it; empty or omitted: the connected context"`
	Section string `json:"section,omitempty" jsonschema:"the section to show: general (name, icon, kubeconfig), proxy, terminal, namespaces (the accessible namespaces), metrics (the Prometheus, hidden metrics), nodeShell (the node shells' image and pull secret) or agents (the allowance of agents' changes, which only the user sets); omitted: general"`
}

const clusterSettingsOnly = " Changes only Cogpit's own cluster settings (a file in Cogpit's configuration directory), never the cluster or its kubeconfig."

// settingsContext is the context a cluster settings command names: the connected one when none
// is given, and a bare name resolved as cluster.connect resolves it; the store must exist.
func settingsContext(s *engine.Session, name string) (string, error) {
	if s.ClusterSettingsStore() == nil {
		return "", engine.ErrNoClusterSettings
	}
	return hotbarContext(s, name)
}

// settingsView is what a cluster settings change returns: the context's settings now.
func settingsView(s *engine.Session, id string) (any, error) {
	return s.ClusterSettingsView(id)
}

// changeSettings resolves the context, checks that a source offers it and changes its settings.
func changeSettings(s *engine.Session, name string, fn func(*engine.ClusterSettings) error) (any, error) {
	id, err := settingsContext(s, name)
	if err != nil {
		return nil, err
	}
	if err := offered(s, id); err != nil {
		return nil, err
	}
	if _, err := s.ChangeClusterSettings(id, fn); err != nil {
		return nil, err
	}
	return settingsView(s, id)
}

func registerClusterSettings(g *registrar) {
	add(g, engine.Spec[clusterSettingsParams]{
		ID: "cluster.settings.get", Label: "Show Cluster Settings", Danger: engine.DangerRead,
		Description: "Returns a context's settings in Cogpit, as its Settings page shows them: the name it shows (the display " +
			"name set, else the context's name), its icon colour or uploaded image's format, the kubeconfig file it comes " +
			"from, its terminals' working directory and kubectl namespace (empty: $HOME and the filter's or the " +
			"context's namespace), the metrics the window hides (hiddenMetrics; empty: all visible), the metrics source (metricsSource: " +
			"auto, prometheus, metrics-server or none) and, while connected, the providers answering (metricsInUse), the accessible " +
			"namespaces named for a user who may not list namespaces (accessibleNamespaces), the node shells' image, pull " +
			"secret and maximum duration (nodeShellImage, empty: nodeShellDefaultImage; nodeShellPullSecret; " +
			"nodeShellMaxDuration, empty: nodeShellDefaultMaxDuration), whether it is " +
			"the connected cluster, whether the user allowed agents' changes without asking (agentWrites), and whether the " +
			"settings are kept. The Prometheus setting is cluster_prometheus.",
		Enabled: func(s *engine.Session, p clusterSettingsParams) error {
			_, err := settingsContext(s, p.Context)
			return err
		},
		Run: func(_ context.Context, s *engine.Session, p clusterSettingsParams) (any, error) {
			id, err := settingsContext(s, p.Context)
			if err != nil {
				return nil, err
			}
			return settingsView(s, id)
		},
	})

	add(g, engine.Spec[clusterSettingsSetParams]{
		ID: "cluster.settings.set", Label: "Set Cluster Settings", Danger: engine.DangerUI,
		Description: "Changes some of a context's settings in Cogpit, as its Settings page does; the fields left out stay as " +
			"they are, an empty value restores the default. The display name and the icon colour show at once in the " +
			"hotbar, the context picker and the navigator; the terminal's working directory and namespace apply to the " +
			"cluster's local terminals opened afterwards; hidden metrics leave the lists' columns, the drawers' charts " +
			"and the Overview at once, and nothing is polled for them; the metrics source turns a provider or both off " +
			"for the cluster at once; a proxy applies at the next connect; accessible " +
			"namespaces (a list that replaces the last) only narrow what Cogpit reads: the namespace filter offers them, " +
			"and a kind the user may not list in all namespaces is watched in each of them, at once. An agent's proxy " +
			"always waits for the user's acceptance, also where the user allowed agent writes; the CLI may clear the proxy " +
			"but not set one; the proxy read back has its password shown as •••." +
			clusterSettingsOnly + " Returns the settings.",
		Enabled: func(s *engine.Session, p clusterSettingsSetParams) error {
			_, err := settingsContext(s, p.Context)
			return err
		},
		Refuse: func(ctx context.Context, _ *engine.Session, p clusterSettingsSetParams) error {
			return refuseProxy(ctx, p.Proxy)
		},
		// A proxy decides where the user's cluster traffic goes, credentials in it: an agent's always
		// waits for the user (A20, owner 2026-10-14 evening).
		AgentsAsk: func(_ *engine.Session, p clusterSettingsSetParams) bool { return setsProxy(p.Proxy) },
		Detail: func(s *engine.Session, p clusterSettingsSetParams) string {
			if !setsProxy(p.Proxy) {
				return ""
			}
			ctx, _ := settingsContext(s, p.Context)
			return proxyDetail("the API traffic of "+ctx+", from its next connect,", *p.Proxy)
		},
		Run: func(ctx context.Context, s *engine.Session, p clusterSettingsSetParams) (any, error) {
			if err := refuseProxy(ctx, p.Proxy); err != nil {
				return nil, err
			}
			var source string
			if p.MetricsSource != nil {
				var err error
				if source, err = engine.ParseMetricsSource(*p.MetricsSource); err != nil {
					return nil, err
				}
			}
			var hidden engine.MetricSet
			if p.HiddenMetrics != nil {
				var err error
				if hidden, err = engine.ParseMetricSet(p.HiddenMetrics); err != nil {
					return nil, err
				}
			}
			return changeSettings(s, p.Context, func(cs *engine.ClusterSettings) error {
				if p.HiddenMetrics != nil {
					cs.HiddenMetrics = hidden
				}
				if p.MetricsSource != nil {
					cs.MetricsSource = source
				}
				if p.AccessibleNamespaces != nil {
					cs.AccessibleNamespaces = slices.Clone(p.AccessibleNamespaces) // the engine trims, sorts and checks them
				}
				setIf(&cs.Name, p.Name)
				if p.IconColor != nil {
					cs.IconColor = strings.ToLower(strings.TrimSpace(*p.IconColor))
					if cs.IconColor != "" {
						cs.Icon = "" // a colour replaces an uploaded image (Lens: Pick Icon Color)
					}
				}
				if p.TerminalDir != nil {
					cs.TerminalDir = strings.TrimSpace(*p.TerminalDir)
				}
				if p.TerminalNamespace != nil {
					cs.TerminalNamespace = strings.TrimSpace(*p.TerminalNamespace)
				}
				if p.Proxy != nil {
					next, err := engine.MergeProxyPassword(cs.Proxy, *p.Proxy) // "•••" keeps the password set before
					if err != nil {
						return err
					}
					cs.Proxy = next
				}
				return nil
			})
		},
	})

	add(g, engine.Spec[clusterIconParams]{
		ID: "cluster.settings.uploadIcon", Label: "Upload Icon", Danger: engine.DangerUI,
		GUIOnly: engine.ErrIconGUIOnly,
		Description: "Makes an image file the user picked a context's icon (it replaces the icon colour): a PNG or JPEG, kept " +
			"as a PNG of at most 256 pixels a side, or an SVG without scripts or outside references, at most 1 MiB. Only the " +
			"user, in Cogpit's window, can upload: refused for agents and the CLI, which can set an icon colour with " +
			"cluster_settings_set." + clusterSettingsOnly + " Returns the settings.",
		Enabled: func(s *engine.Session, p clusterIconParams) error {
			_, err := settingsContext(s, p.Context)
			return err
		},
		Refuse: func(ctx context.Context, _ *engine.Session, _ clusterIconParams) error {
			if engine.OriginOf(ctx) != engine.OriginGUI {
				return engine.ErrIconGUIOnly
			}
			return nil
		},
		Run: func(ctx context.Context, s *engine.Session, p clusterIconParams) (any, error) {
			if engine.OriginOf(ctx) != engine.OriginGUI {
				return nil, engine.ErrIconGUIOnly
			}
			id, err := settingsContext(s, p.Context)
			if err != nil {
				return nil, err
			}
			if err := offered(s, id); err != nil {
				return nil, err
			}
			if _, err := s.SetClusterIcon(id, p.Path); err != nil {
				return nil, err
			}
			return settingsView(s, id)
		},
	})

	add(g, engine.Spec[clusterSettingsParams]{
		ID: "cluster.settings.resetIcon", Label: "Reset Icon", Danger: engine.DangerUI,
		Description: "Gives a context its default icon back: its initials in the accent colour, without an uploaded image or " +
			"an icon colour." + clusterSettingsOnly + " Returns the settings.",
		Enabled: func(s *engine.Session, p clusterSettingsParams) error {
			_, err := settingsContext(s, p.Context)
			return err
		},
		Run: func(_ context.Context, s *engine.Session, p clusterSettingsParams) (any, error) {
			return changeSettings(s, p.Context, func(cs *engine.ClusterSettings) error {
				cs.Icon, cs.IconColor = "", ""
				return nil
			})
		},
	})

	add(g, engine.Spec[clusterSettingsOpenParams]{
		ID: "cluster.settings.open", Label: "Cluster Settings", MenuPath: "Cluster/Settings", Shortcut: "Cmd+Shift+,", Danger: engine.DangerUI,
		Description: "Opens a context's Settings page in Cogpit's window, as ⌘⇧, (Ctrl+Shift+,) does for the connected cluster, " +
			"at a section: general (name, icon, kubeconfig), proxy, terminal, namespaces (the accessible namespaces), metrics (the Prometheus, hidden metrics), nodeShell or agents. It takes the place of the cluster's views " +
			"until closed (settings_close, Escape); ui_state reports it as settings. Returns the new UI state.",
		Enabled: func(s *engine.Session, p clusterSettingsOpenParams) error {
			if _, err := settingsContext(s, p.Context); err != nil {
				return err
			}
			_, err := engine.CheckSection(engine.SettingsCluster, p.Section)
			return err
		},
		Run: func(_ context.Context, s *engine.Session, p clusterSettingsOpenParams) (any, error) {
			id, err := settingsContext(s, p.Context)
			if err != nil {
				return nil, err
			}
			if err := offered(s, id); err != nil {
				return nil, err
			}
			section, err := engine.CheckSection(engine.SettingsCluster, p.Section)
			if err != nil {
				return nil, err
			}
			return s.UpdateUI(func(u *engine.UIState) {
				u.Settings = &engine.SettingsPage{Page: engine.SettingsCluster, Context: id, Section: section}
				u.Palette = false
			}), nil
		},
	})
}
