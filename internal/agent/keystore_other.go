//go:build !(darwin && cgo)

package agent

// NewSystemKeyStore returns the operating system's keychain. Cogpit has none here yet (Windows and
// Linux are M11): every Set fails with ErrNoKeychain, so a connection that needs a key cannot be
// saved with one, and nothing is ever written to a file instead.
func NewSystemKeyStore() KeyStore { return unavailableKeyStore{} }
