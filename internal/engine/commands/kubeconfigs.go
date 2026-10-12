package commands

import (
	"context"
	"errors"
	"strings"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// The kubeconfig sources (G2): add a kubeconfig file, add a sync folder,
// remove a source, and allow a source context's credential plugin before its first run
// (trust-boundaries H-1). Reading the sources and a plugin's description is open to every origin;
// everything that adds what Cogpit may connect through, or lets a program run, is the user's own
// action in Cogpit's window only (engine.ErrGUIOnly), refused for agents and the CLI whatever they
// pass: a kubeconfig can name a program to run on this computer with the user's rights.

// SourcesResult is kubeconfig.sources' result.
type SourcesResult struct {
	Sources []engine.SourceStatus `json:"sources"`
}

// SourceAdded is the result of the commands that add a source: the source and the ids of the
// contexts it offers now.
type SourceAdded struct {
	Source   engine.KubeconfigSource `json:"source"`
	Contexts []string                `json:"contexts"`
}

// ExecPluginResult is kubeconfig.execPlugin's result. Agents get the plugin without its arguments
// (ArgsWithheld counts them): an argument may carry a credential, and the user sees them in
// Cogpit's window.
type ExecPluginResult struct {
	engine.ExecPluginInfo
	ArgsWithheld int `json:"argsWithheld,omitempty"`
}

type pathParams struct {
	Path string `json:"path" jsonschema:"absolute path of the file or folder"`
}

type sourceParams struct {
	Source string `json:"source" jsonschema:"source id as kubeconfig_sources lists it"`
}

type execParams struct {
	Context string `json:"context" jsonschema:"context id of a kubeconfig source as cluster_contexts lists it: <context> [<source>]"`
}

type approveParams struct {
	Context     string `json:"context" jsonschema:"context id of a kubeconfig source as cluster_contexts lists it"`
	Fingerprint string `json:"fingerprint" jsonschema:"the plugin's fingerprint as kubeconfig_execPlugin returned it, binding the approval to what the user was shown"`
}

// guiOnly is the note every GUI-only command's description ends with.
const guiOnly = " Only the user's own action in Cogpit's window runs it: agents and the CLI are refused, " +
	"because a kubeconfig can name a program to run on this computer."

// sourcesAvailable is the Enabled check of the kubeconfig commands.
func sourcesAvailable(s *engine.Session) (*engine.KubeconfigSources, error) {
	k := s.KubeconfigSources()
	if k == nil {
		return nil, errors.New("this Cogpit keeps no kubeconfig sources")
	}
	return k, nil
}

// requireGUI refuses every origin but the GUI.
func requireGUI(ctx context.Context) error {
	if engine.OriginOf(ctx) != engine.OriginGUI {
		return engine.ErrGUIOnly
	}
	return nil
}

// added emits the change and returns what a source offers now.
func added(s *engine.Session, src engine.KubeconfigSource) SourceAdded {
	s.Emit(engine.Event{Type: engine.EventContexts})
	out := SourceAdded{Source: src, Contexts: []string{}}
	ctxs, _ := s.KubeconfigSources().Contexts()
	for _, c := range ctxs {
		if c.Source == src.ID || (src.Kind == engine.SourceFolder && strings.HasPrefix(c.Source, src.ID)) {
			out.Contexts = append(out.Contexts, c.Name)
		}
	}
	return out
}

func registerKubeconfigs(g *registrar) {
	enabled := func(s *engine.Session) error { _, err := sourcesAvailable(s); return err }

	add(g, engine.Spec[struct{}]{
		ID: "kubeconfig.sources", Label: "Kubeconfig Sources", MenuPath: "File/Kubeconfig Sources", Danger: engine.DangerRead,
		Description: "Lists the kubeconfig sources the user added in Cogpit: kubeconfig files and sync folders, each with its id, kind, path, the files read from it, how many contexts it " +
			"offers and why it cannot be read. Their contexts appear in cluster_contexts as \"<context> [<source>]\". Reads " +
			"only file names and counts; no credential is returned.",
		Enabled: func(s *engine.Session, _ struct{}) error { return enabled(s) },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			k, err := sourcesAvailable(s)
			if err != nil {
				return nil, err
			}
			_, statuses, err := k.Files()
			if err != nil {
				return nil, err
			}
			if statuses == nil {
				statuses = []engine.SourceStatus{}
			}
			return SourcesResult{Sources: statuses}, nil
		},
	})

	add(g, engine.Spec[pathParams]{
		ID: "kubeconfig.addFile", Label: "Add Kubeconfig File", MenuPath: "File/Add Kubeconfig File", Danger: engine.DangerUI,
		Description: "Adds a kubeconfig file the user picked as a source: its contexts become available as " +
			"\"<context> [<file name>]\", read from that file alone, and follow its changes. Cogpit reads the file and never " +
			"writes it. A credential plugin the file names runs only after the user allowed it." + guiOnly,
		Enabled: func(s *engine.Session, _ pathParams) error { return enabled(s) },
		Run: func(ctx context.Context, s *engine.Session, p pathParams) (any, error) {
			if err := requireGUI(ctx); err != nil {
				return nil, err
			}
			k, err := sourcesAvailable(s)
			if err != nil {
				return nil, err
			}
			src, _, err := k.AddFile(p.Path)
			if err != nil {
				return nil, err
			}
			return added(s, src), nil
		},
	})

	add(g, engine.Spec[pathParams]{
		ID: "kubeconfig.addFolder", Label: "Add Sync Folder", MenuPath: "File/Add Sync Folder", Danger: engine.DangerUI,
		Description: "Adds a sync folder: every kubeconfig file in it (not in its subfolders) becomes available as " +
			"\"<context> [<folder>/<file>]\", and files that appear in it or leave it later are picked up or dropped. " +
			"Cogpit reads the files and never writes them. A credential plugin a file names runs only after the user " +
			"allowed it." + guiOnly,
		Enabled: func(s *engine.Session, _ pathParams) error { return enabled(s) },
		Run: func(ctx context.Context, s *engine.Session, p pathParams) (any, error) {
			if err := requireGUI(ctx); err != nil {
				return nil, err
			}
			k, err := sourcesAvailable(s)
			if err != nil {
				return nil, err
			}
			src, err := k.AddFolder(p.Path)
			if err != nil {
				return nil, err
			}
			return added(s, src), nil
		},
	})

	add(g, engine.Spec[sourceParams]{
		ID: "kubeconfig.remove", Label: "Remove Kubeconfig Source", MenuPath: "File/Remove Kubeconfig Source", Danger: engine.DangerUI,
		Description: "Removes a kubeconfig source, which is Cogpit's link to a file or folder and nothing else: its contexts " +
			"are no longer offered, the credential plugins allowed for its files are forgotten, and an open connection made " +
			"from one of its files is closed. The file or folder the user added, and the files in it, stay on disk as they are. " +
			"The GUI asks the user first, naming the path." + guiOnly,
		Enabled: func(s *engine.Session, _ sourceParams) error { return enabled(s) },
		Run: func(ctx context.Context, s *engine.Session, p sourceParams) (any, error) {
			if err := requireGUI(ctx); err != nil {
				return nil, err
			}
			k, err := sourcesAvailable(s)
			if err != nil {
				return nil, err
			}
			src, err := k.Remove(p.Source)
			if src.ID != "" {
				s.CloseGoneSourceConnection() // a connection made from the source's files ends with the link
				s.Emit(engine.Event{Type: engine.EventContexts})
			}
			if err != nil {
				return nil, err
			}
			return src, nil
		},
	})

	add(g, engine.Spec[execParams]{
		ID: "kubeconfig.execPlugin", Label: "Show Credential Plugin", Danger: engine.DangerRead,
		Description: "Describes how a kubeconfig source's context logs in when that needs the user's approval: the credential " +
			"plugin (users[].user.exec) it runs and a token file (users[].user.tokenFile) outside the kubeconfig's folder. " +
			"Shows the command, the kubeconfig file it comes from, the API server the credential goes to and how that " +
			"server is reached and trusted (proxy URL, insecure-skip-tls-verify, TLS server name, own CA), the " +
			"SHA-256 of a program or CA file inside the kubeconfig's folder, the names of the environment variables it " +
			"sets, and whether the user allowed it (state approved, required or changed). " +
			"Cogpit runs such a plugin only once the user allowed it in Cogpit's window; agents cannot allow it. Agents " +
			"get the argument count, not the arguments. Runs nothing.",
		Enabled: func(s *engine.Session, _ execParams) error { return enabled(s) },
		Run: func(ctx context.Context, s *engine.Session, p execParams) (any, error) {
			info, err := s.ExecPlugin(p.Context)
			if err != nil {
				return nil, err
			}
			out := ExecPluginResult{ExecPluginInfo: *info}
			if engine.OriginOf(ctx) != engine.OriginGUI && len(info.Plugin.Args) > 0 {
				plugin := *info.Plugin
				out.ArgsWithheld, plugin.Args = len(plugin.Args), nil
				out.Plugin = &plugin
			}
			return out, nil
		},
	})

	add(g, engine.Spec[approveParams]{
		ID: "kubeconfig.approveExec", Label: "Allow Credential Plugin", Danger: engine.DangerUI,
		Description: "Records that the user allows a kubeconfig source context's credential plugin to run, exactly as shown " +
			"(the fingerprint binds the command, its arguments, its environment and the server); a later change asks again." +
			guiOnly,
		Enabled: func(s *engine.Session, _ approveParams) error { return enabled(s) },
		Run: func(ctx context.Context, s *engine.Session, p approveParams) (any, error) {
			if err := requireGUI(ctx); err != nil {
				return nil, err
			}
			return s.ApproveExec(p.Context, p.Fingerprint)
		},
	})
}
