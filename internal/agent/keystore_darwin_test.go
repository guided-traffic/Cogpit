//go:build darwin && cgo

package agent

import (
	"errors"
	"os"
	"testing"
)

// TestMacKeyStoreRoundTrip stores, reads, replaces and removes a key in the login keychain under a
// test service of its own, never Cogpit's. It writes to the user's keychain, so it runs only with
// COGPIT_KEYCHAIN_TEST=1 (CI's macOS runner may have no unlocked keychain).
func TestMacKeyStoreRoundTrip(t *testing.T) {
	if os.Getenv("COGPIT_KEYCHAIN_TEST") != "1" {
		t.Skip("set COGPIT_KEYCHAIN_TEST=1 to write a test item to the login keychain")
	}
	k := macKeyStore{service: KeyService + ".test"}
	const account = "cogpit-keychain-test"
	t.Cleanup(func() { _ = k.Delete(account) })
	if _, err := k.Get(account); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Get before Set: %v, want ErrNoKey", err)
	}
	for _, want := range []string{"first-test-value-ä", "second-test-value"} {
		if err := k.Set(account, want); err != nil {
			t.Fatal(err)
		}
		got, err := k.Get(account)
		if err != nil || got != want {
			t.Fatalf("Get = %q, %v; want the value set", got, err)
		}
	}
	if err := k.Delete(account); err != nil {
		t.Fatal(err)
	}
	if _, err := k.Get(account); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Get after Delete: %v, want ErrNoKey", err)
	}
	if err := k.Delete(account); err != nil {
		t.Fatalf("Delete of a missing item: %v", err)
	}
}
