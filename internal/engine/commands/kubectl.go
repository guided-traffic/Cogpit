package commands

import (
	"context"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// kubectl in terminals (D-3; engine/kubectl.go). A terminal's tab offers the kubectl release that
// matches its cluster when the one it finds is more than one minor version away (UIState: the
// tab's kubectl). kubectl.download fetches it, verified, into the directory for binaries. It puts
// a program first on the PATH of Cogpit's terminals, so only the user, in Cogpit's window, runs
// it: agents and the CLI see the offer in ui_state and are refused (engine.ErrKubectlGUIOnly).

// KubectlDownloadResult is what kubectl.download returns.
type KubectlDownloadResult struct {
	// Path is the verified binary in the directory for binaries.
	Path string `json:"path"`
}

type kubectlDownloadParams struct {
	Tab string `json:"tab" jsonschema:"the terminal dock tab whose offer to download, as ui_state lists it (its kubectl.offer)"`
}

func registerKubectl(g *registrar) {
	add(g, engine.Spec[kubectlDownloadParams]{
		ID: "kubectl.download", Label: "Download kubectl", Danger: engine.DangerUI,
		GUIOnly: engine.ErrKubectlGUIOnly,
		Description: "Downloads the kubectl release a terminal tab offers (the cluster's version, when the kubectl the " +
			"terminal finds is more than one minor version away or missing) from dl.k8s.io or the mirror set in " +
			"Preferences, verifies it against the published SHA-256, the Kubernetes release's signature and that signature's " +
			"entry in sigstore's transparency log (rekor.sigstore.dev), and keeps it " +
			"in the directory for binaries; terminals opened afterwards find it first on PATH. Only the user, in Cogpit's " +
			"window, can download: refused for agents and the CLI, which see the offer in ui_state. Changes no cluster " +
			"and no kubeconfig. Returns the binary's path.",
		Enabled: func(s *engine.Session, p kubectlDownloadParams) error { return s.KubectlDownloadable(p.Tab) },
		Refuse: func(ctx context.Context, _ *engine.Session, _ kubectlDownloadParams) error {
			if engine.OriginOf(ctx) != engine.OriginGUI {
				return engine.ErrKubectlGUIOnly
			}
			return nil
		},
		Run: func(ctx context.Context, s *engine.Session, p kubectlDownloadParams) (any, error) {
			if engine.OriginOf(ctx) != engine.OriginGUI {
				return nil, engine.ErrKubectlGUIOnly
			}
			path, err := s.DownloadKubectl(ctx, p.Tab)
			if err != nil {
				return nil, err
			}
			return KubectlDownloadResult{Path: path}, nil
		},
	})
}
