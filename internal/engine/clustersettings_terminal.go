package engine

import (
	"fmt"
	"os"
	"slices"

	"github.com/guided-traffic/cogpit/internal/terminal"
)

// LocalTerminalOptions completes the options of a local terminal bound to a context with the
// settings, when the terminal starts: the Preferences' shell and its arguments, and the context's
// Cluster settings › Terminal: the working directory (else the home directory, the terminal's
// default) and kubectl's namespace, which, when set, replaces the namespace filter's and the
// context's own (Cogpit's default, inferred: the setting is the user's explicit choice for that
// cluster). A working directory that does not exist is an error the terminal's tab shows.
func (s *Session) LocalTerminalOptions(context string, opts terminal.LocalOptions) (terminal.LocalOptions, error) {
	if p, err := s.Preferences(); err == nil {
		opts.Shell, opts.Args = p.TerminalShell, slices.Clone(p.TerminalShellArgs)
	}
	cs, err := s.ClusterSettingsOf(context)
	if err != nil {
		return opts, nil // a session without cluster settings: the defaults
	}
	if cs.TerminalDir != "" {
		dir, err := ExpandTerminalDir(cs.TerminalDir)
		if err != nil {
			return opts, err
		}
		if fi, err := os.Stat(dir); err != nil || !fi.IsDir() {
			return opts, fmt.Errorf("the working directory %s of this cluster's terminals is not a folder (Cluster Settings › Terminal)", dir)
		}
		opts.Dir = dir
	}
	if cs.TerminalNamespace != "" {
		opts.Namespace = cs.TerminalNamespace
	}
	return opts, nil
}
