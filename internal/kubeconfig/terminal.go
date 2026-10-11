package kubeconfig

// TerminalConfig returns the kubeconfig a terminal puts first on KUBECONFIG: only a
// current-context and one context entry that names the cluster and user entries of the user's own
// files, plus the namespace. It holds no credential (docs/security/credentials.md).
func TerminalConfig(ctx Context) ([]byte, error) {
	_ = ctx
	return nil, ErrNotImplemented
}
