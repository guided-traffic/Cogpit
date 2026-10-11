// Package terminal runs interactive sessions for dock tabs: a local shell on Cogpit's own PTY with
// a per-terminal kubeconfig, and a shell in a pod through client-go's exec stream.
//
// Layer L1. Terminal output is untrusted: it comes from the shell or the container.
package terminal

import (
	"context"
	"errors"
	"io"

	_ "github.com/creack/pty" // pinned; Phase 1 uses it

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ErrNotImplemented marks a Phase 0 stub.
var ErrNotImplemented = errors.New("not implemented")

// Session is one running terminal session. Read returns the session's output, Write sends the
// user's keystrokes. Close ends the session and frees its resources (for a local shell: the PTY,
// the process and the temporary kubeconfig file).
type Session interface {
	io.ReadWriteCloser
	// Resize tells the session the terminal's size in character cells.
	Resize(cols, rows uint16) error
	// Done is closed when the session has ended on its own (the shell exited, the stream closed).
	Done() <-chan struct{}
}

// LocalOptions configure a local shell.
type LocalOptions struct {
	// Shell is the shell to run; empty means $SHELL, else /bin/zsh. It runs as a login shell.
	Shell string
	// Dir is the working directory; empty means the user's home directory.
	Dir string
	// KubeconfigPaths are the user's kubeconfig files in precedence order. The session's own
	// kubeconfig goes in front of them on KUBECONFIG.
	KubeconfigPaths []string
	// Context and Namespace bind kubectl in the shell to one context and namespace.
	Context   string
	Namespace string
	// Env is added to the environment.
	Env []string
}

// StartLocal starts a local shell on a new PTY with KUBECONFIG=<own file>:<the user's files>,
// where the own file holds only the context and namespace (docs/security/credentials.md). The
// file is owner-only and removed when the session closes.
func StartLocal(opts LocalOptions) (Session, error) {
	_ = opts
	return nil, ErrNotImplemented
}

// ExecOptions select a container to run a shell in.
type ExecOptions struct {
	Namespace string
	Pod       string
	Container string   // empty: the pod's default container
	Command   []string // empty: try bash, then ash, then sh
}

// StartPodExec opens an interactive shell in a container through the API server's exec
// subresource, with a TTY and resize support.
func StartPodExec(ctx context.Context, config *rest.Config, client kubernetes.Interface, opts ExecOptions) (Session, error) {
	_, _, _, _ = ctx, config, client, opts
	return nil, ErrNotImplemented
}
