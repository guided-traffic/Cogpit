// Package app is the wiring that the desktop app (cmd/cogpit) and the command-line tool
// (cmd/cogpit-cli) share: where Cogpit keeps its own files, which kubeconfig sources a process
// offers, and the audit log. Both binaries build them here, so that they offer the same contexts
// under the same ids and run a credential plugin only once the user allowed it in the same
// approvals file (trust-boundaries H-1).
//
// Layer L4. It imports the engine and nothing that draws a window, serves MCP or talks to an API
// server.
package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Names under Cogpit's configuration directory.
const (
	// DirName is Cogpit's directory under the user's configuration directory.
	DirName = "Cogpit"
	// AuditLogFile is the audit log of agent-initiated commands (D-2).
	AuditLogFile = "audit.log"
)

// Dirs are the directories a Cogpit process reads and writes.
type Dirs struct {
	// Config is Cogpit's own directory, <user config dir>/Cogpit: the kubeconfig sources and
	// approvals, the journals, the hotbar, the audit log and the MCP token live there.
	Config string
	// Home is the user's home directory, whose .kube is the default sync folder (D-28); empty
	// means none.
	Home string
}

// UserDirs returns the directories of the user running the process. Without a configuration
// directory there is nothing to return; without a home directory Home stays empty.
func UserDirs() (Dirs, error) {
	cfg, err := os.UserConfigDir()
	if err != nil {
		return Dirs{}, err
	}
	d := Dirs{Config: filepath.Join(cfg, DirName)}
	if home, err := os.UserHomeDir(); err == nil {
		d.Home = home
	}
	return d, nil
}

// KubeconfigSources returns the kubeconfig sources a process offers, kept in d.Config
// (kubeconfigs.json, exec-approvals.json, each owner-only): the user's own ~/.kube as a sync
// folder from the start, as in Lens, removable like any other (none without a home directory);
// the files named for this run (named: --kubeconfig, else $KUBECONFIG; cluster.NamedKubeconfigFiles)
// as sources of their own, removable for the session; and the files and sync folders the user
// added. Every context comes from a source file read on its own; nothing is merged (D-28). Cogpit
// reads these files, never writes them, and a credential plugin of any of them needs the user's
// approval (trust-boundaries H-1). Nothing is read until the sources are used.
func (d Dirs) KubeconfigSources(named []string) *engine.KubeconfigSources {
	opts := []engine.SourcesOption{engine.WithSessionFiles(named)}
	if d.Home != "" {
		opts = append(opts, engine.WithDefaultFolder(filepath.Join(d.Home, ".kube")))
	}
	return engine.NewKubeconfigSources(d.Config, opts...)
}

// AuditLogPath is the audit log's file, d.Config/audit.log, which the Audit Log reads back.
func (d Dirs) AuditLogPath() string {
	if d.Config == "" {
		return ""
	}
	return filepath.Join(d.Config, AuditLogFile)
}

// OpenAuditLog opens d.Config/audit.log for appending, owner-only (0600 in a 0700 directory), for
// the commands agents dispatch (D-2). The caller closes it.
func (d Dirs) OpenAuditLog() (io.WriteCloser, error) {
	if d.Config == "" {
		return nil, errors.New("audit log: no configuration directory")
	}
	if err := os.MkdirAll(d.Config, 0o700); err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}
	f, err := os.OpenFile(filepath.Join(d.Config, AuditLogFile), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("audit log: %w", err)
	}
	return f, nil
}
