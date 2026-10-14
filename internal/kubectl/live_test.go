package kubectl

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestLiveRelease verifies the current stable kubectl release's signature for this platform
// against sigstore's public transparency log, online, as a download does before it asks for the
// binary: the .sha256, .sig and .cert from dl.k8s.io, the entry from rekor.sigstore.dev, and the
// embedded trusted root. It runs only with COGPIT_SIGSTORE_LIVE=1, never in CI; with
// COGPIT_SIGSTORE_LIVE=download it then downloads the release (about 57 MB) into a temporary folder.
//
//	COGPIT_SIGSTORE_LIVE=1 go test ./internal/kubectl -run TestLiveRelease -v
func TestLiveRelease(t *testing.T) {
	live := os.Getenv("COGPIT_SIGSTORE_LIVE")
	if live != "1" && live != "download" {
		t.Skip("reaches dl.k8s.io and rekor.sigstore.dev: set COGPIT_SIGSTORE_LIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	d := &Downloader{}
	client := d.client()
	stable, err := fetchSmall(ctx, client, DefaultMirror+"/stable.txt")
	if err != nil {
		t.Fatal(err)
	}
	v, err := ParseVersion(string(stable))
	if err != nil {
		t.Fatal(err)
	}
	u, err := d.URL(v)
	if err != nil {
		t.Fatal(err)
	}
	read := func(ext string) []byte {
		data, err := fetchSmall(ctx, client, u+ext)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	digest, err := parseSHA256(read(".sha256"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := ParseSigned(digest, read(".sig"), read(".cert"))
	if err != nil {
		t.Fatal(err)
	}
	entries, err := lookupRekor(ctx, client, DefaultRekor, s)
	if err != nil {
		t.Fatal(err)
	}
	if err := DefaultVerifier().Verify(s, entries); err != nil {
		t.Fatalf("kubectl %s (%s): %v", v, u, err)
	}
	t.Logf("kubectl %s (%s): verified, Rekor entry %d of %s", v, u, entries[0].LogIndex,
		time.Unix(entries[0].IntegratedTime, 0).UTC().Format(time.RFC3339))
	if live != "download" {
		return
	}
	path, err := d.Download(ctx, v, t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("downloaded and verified into %s", path)
}
