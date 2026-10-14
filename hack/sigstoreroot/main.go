// Command sigstoreroot refreshes the sigstore trusted root that internal/kubectl embeds
// (sigstore_trusted_root.json): the public-good instance's Fulcio certificate authorities, Rekor
// logs, certificate transparency logs and timestamp authorities, with their keys and validity
// periods (docs/developer/decisions.md D-50). It fetches trusted_root.json through sigstore's TUF
// repository (https://tuf-repo-cdn.sigstore.dev), verified from the TUF root that sigstore-go
// embeds, with a cache of its own in a temporary folder that it removes; it checks that the file
// parses as a trusted root and names a Rekor log before it writes it. Renovate cannot see the file:
// run it before a release, and when a kubectl download says the signing certificate or the log
// entry is not one the trusted root knows.
//
//	make sigstore-root   # go run ./hack/sigstoreroot -o internal/kubectl/sigstore_trusted_root.json
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"

	"github.com/sigstore/sigstore-go/pkg/root"
	"github.com/sigstore/sigstore-go/pkg/tuf"
)

func main() {
	out := flag.String("o", "internal/kubectl/sigstore_trusted_root.json", "the file to write")
	flag.Parse()
	if err := run(*out); err != nil {
		fmt.Fprintln(os.Stderr, "sigstoreroot:", err)
		os.Exit(1)
	}
}

func run(out string) error {
	cache, err := os.MkdirTemp("", "cogpit-sigstore-tuf-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(cache) }()
	client, err := tuf.New(tuf.DefaultOptions().WithCachePath(cache))
	if err != nil {
		return fmt.Errorf("sigstore's TUF repository: %w", err)
	}
	data, err := client.GetTarget("trusted_root.json")
	if err != nil {
		return fmt.Errorf("trusted_root.json from sigstore's TUF repository: %w", err)
	}
	tr, err := root.NewTrustedRootFromJSON(data)
	if err != nil {
		return fmt.Errorf("trusted_root.json does not parse: %w", err)
	}
	if len(tr.RekorLogs()) == 0 || len(tr.FulcioCertificateAuthorities()) == 0 || len(tr.CTLogs()) == 0 {
		return fmt.Errorf("trusted_root.json names %d Rekor logs, %d certificate authorities and %d CT logs; want at least one of each",
			len(tr.RekorLogs()), len(tr.FulcioCertificateAuthorities()), len(tr.CTLogs()))
	}
	if old, err := os.ReadFile(out); err == nil && bytes.Equal(old, data) {
		fmt.Println(out, "is current")
		return nil
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote %s (%d bytes)\n", out, len(data))
	return nil
}
