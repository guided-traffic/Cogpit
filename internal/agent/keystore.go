package agent

import (
	"errors"
	"sync"
)

// KeyService is the keychain service under which Cogpit keeps the API keys of its LLM
// connections, one generic password per connection, the connection's id the account
// (docs/security/credentials.md).
const KeyService = "io.guided-traffic.cogpit.llm"

// ErrNoKey is returned by KeyStore.Get when no key is stored for the account.
var ErrNoKey = errors.New("no API key in the keychain for this connection")

// ErrNoKeychain is the platform's answer where Cogpit has no keychain yet (Windows and Linux: M11).
var ErrNoKeychain = errors.New("no keychain on this platform yet: Cogpit cannot keep an API key here")

// KeyStore keeps the API keys of the LLM connections outside Cogpit's files: the operating
// system's keychain (NewSystemKeyStore), or memory in tests. Errors never carry a key.
type KeyStore interface {
	// Get returns the key stored for account, ErrNoKey when there is none.
	Get(account string) (string, error)
	// Set stores key for account, replacing one stored before.
	Set(account, key string) error
	// Delete removes the key of account; nil when there was none.
	Delete(account string) error
}

// MemoryKeyStore is a KeyStore in memory, for tests.
type MemoryKeyStore struct {
	mu   sync.Mutex
	keys map[string]string
}

// NewMemoryKeyStore returns an empty MemoryKeyStore.
func NewMemoryKeyStore() *MemoryKeyStore { return &MemoryKeyStore{keys: map[string]string{}} }

// Get implements KeyStore.
func (m *MemoryKeyStore) Get(account string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	k, ok := m.keys[account]
	if !ok {
		return "", ErrNoKey
	}
	return k, nil
}

// Set implements KeyStore.
func (m *MemoryKeyStore) Set(account, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.keys[account] = key
	return nil
}

// Delete implements KeyStore.
func (m *MemoryKeyStore) Delete(account string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.keys, account)
	return nil
}

// Len returns how many keys are stored.
func (m *MemoryKeyStore) Len() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.keys)
}

// unavailableKeyStore is the KeyStore of a platform without a keychain implementation.
type unavailableKeyStore struct{}

func (unavailableKeyStore) Get(string) (string, error) { return "", ErrNoKeychain }
func (unavailableKeyStore) Set(string, string) error   { return ErrNoKeychain }
func (unavailableKeyStore) Delete(string) error        { return nil } // nothing can be stored, so nothing is there
