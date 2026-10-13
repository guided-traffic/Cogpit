package engine

// What Cogpit shows for a context whose kubeconfig switches the check of the API server's
// certificate off (insecure-skip-tls-verify, H-37 in docs/security/trust-boundaries.md): client-go
// connects without asking, as kubectl does, and sends the context's credential to whoever answers
// at the server's address. Lens's documentation says nothing about such contexts; Cogpit's
// default (2026-10-13, not the owner's decision) is to connect, as kubectl does, and to say so in
// two places: the picker marks the context (ContextInfo.Insecure, from the kubeconfig) and the
// connect log gets a WARN line once the connection stands (from the connection's own
// configuration, so it is what client-go actually does).

// insecureLogLine is the connect log's warning for a connection that does not check the server's
// certificate; "" for any other.
func insecureLogLine(b *Backend) string {
	if b == nil || b.Conn == nil || !b.Conn.Insecure() {
		return ""
	}
	return "Certificate checks are off for this context (insecure-skip-tls-verify in its kubeconfig): " +
		"Cogpit cannot tell the cluster at " + b.Conn.Server() + " from anyone else who answers there, " +
		"and sends the context's credential to whoever does"
}
