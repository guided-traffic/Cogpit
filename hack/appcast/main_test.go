package main

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const placeholder = "COGPIT-DMG-SIGNATURE"

// testKey is a throwaway Ed25519 key and its SUPublicEDKey.
func testKey(t *testing.T) (ed25519.PrivateKey, string) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	return priv, base64.StdEncoding.EncodeToString(pub)
}

// signFeed signs a feed as Sparkle's sign_update does: the content, then the signing block.
func signFeed(priv ed25519.PrivateKey, content []byte) []byte {
	sig := ed25519.Sign(priv, content)
	block := fmt.Sprintf("%sedSignature: %s\nlength: %d\n%s\n", signingPrefix, base64.StdEncoding.EncodeToString(sig), len(content), signingSuffix)
	return append(bytes.Clone(content), block...)
}

// fill replaces the draft's placeholder with sig, as the signing job does.
func fill(draft []byte, sig []byte) []byte {
	return bytes.Replace(draft, []byte(placeholder), []byte(base64.StdEncoding.EncodeToString(sig)), 1)
}

func baseOptions(t *testing.T, pub, version string) addOptions {
	t.Helper()
	dir := t.TempDir()
	return addOptions{pubkey: pub, out: filepath.Join(dir, "appcast.xml"), version: version, short: version, min: "13.0",
		url:    "https://github.com/guided-traffic/cogpit/releases/download/v" + version + "/Cogpit-" + version + "-macos-arm64.dmg",
		length: 1234, signature: placeholder, link: "https://github.com/guided-traffic/cogpit/releases/tag/v" + version,
		published: "2026-10-14T12:00:00Z", keep: 10}
}

// The first release starts the feed: one item with the release's versions, the minimum system
// version in three parts, the disk image's URL and length, the placeholder for its signature, the
// release notes up to the pipeline's note, and the links to the release; the draft parses and
// renders the same again.
func TestAddStartsTheFeed(t *testing.T) {
	_, pub := testKey(t)
	o := baseOptions(t, pub, "0.5.0")
	o.feed = filepath.Join(t.TempDir(), "missing.xml")
	o.notes = filepath.Join(t.TempDir(), "notes.md")
	notes := "## [0.5.0](https://example.invalid) (2026-10-14)\n\n### Features\n\n* updates ]]> safely\n\n---\n\n**Downloads.** the pipeline's note\n"
	if err := os.WriteFile(o.notes, []byte(notes), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := add(o)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parse(data)
	if err != nil || len(f.Items) != 1 {
		t.Fatalf("draft %s: %v", data, err)
	}
	it := f.Items[0]
	want := item{Title: "Cogpit 0.5.0", Link: o.link, Version: "0.5.0", ShortVersion: "0.5.0", MinimumSystemVersion: "13.0.0",
		FullReleaseNotesLink: "https://github.com/guided-traffic/cogpit/releases", PubDate: "Wed, 14 Oct 2026 12:00:00 +0000",
		Description: description{Format: "markdown", Text: "## [0.5.0](https://example.invalid) (2026-10-14)\n\n### Features\n\n* updates ]]> safely\n\n[The release on GitHub](" + o.link + ")"},
		Enclosure:   enclosure{URL: o.url, Length: 1234, Type: "application/octet-stream", EdSignature: placeholder}}
	if it != want {
		t.Errorf("item\n got %+v\nwant %+v", it, want)
	}
	if bytes.Count(data, []byte(placeholder)) != 1 || bytes.Contains(data, []byte("Downloads.")) {
		t.Errorf("draft %s", data)
	}
	again, err := render(f.Items)
	if err != nil || !bytes.Equal(again, data) {
		t.Errorf("rendered again differs (%v):\n%s\n%s", err, again, data)
	}
}

// add builds on a feed signed with the app's key only; an unsigned, altered or foreign-signed
// feed is refused, so its history is never signed again.
func TestAddRefusesAnUnsignedHistory(t *testing.T) {
	priv, pub := testKey(t)
	other, _ := testKey(t)
	first := baseOptions(t, pub, "0.5.0")
	draft, err := add(first)
	if err != nil {
		t.Fatal(err)
	}
	content := fill(draft, ed25519.Sign(priv, []byte("dmg")))
	for name, data := range map[string][]byte{
		"unsigned":       content,
		"other key":      signFeed(other, content),
		"altered":        bytes.Replace(signFeed(priv, content), []byte("Cogpit 0.5.0"), []byte("Cogpit 0.5.1"), 1),
		"appended after": append(signFeed(priv, content), "<rss/>"...),
	} {
		t.Run(name, func(t *testing.T) {
			o := baseOptions(t, pub, "0.6.0")
			o.feed = filepath.Join(t.TempDir(), "appcast.xml")
			if err := os.WriteFile(o.feed, data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := add(o); err == nil || !strings.Contains(err.Error(), "refusing") {
				t.Errorf("built on it: %v", err)
			}
		})
	}
}

// A signed history keeps its items: a release of the same version replaces its item, the items
// are newest first by their versions' numbers (0.10.0 after 0.9.0), and each channel keeps its
// newest.
func TestAddMergesTheHistory(t *testing.T) {
	priv, pub := testKey(t)
	var feed []byte
	publish := func(version, channel string, keep int) {
		t.Helper()
		o := baseOptions(t, pub, version)
		o.channel, o.keep = channel, keep
		if feed != nil {
			o.feed = filepath.Join(t.TempDir(), "appcast.xml")
			if err := os.WriteFile(o.feed, feed, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		draft, err := add(o)
		if err != nil {
			t.Fatalf("%s: %v", version, err)
		}
		feed = signFeed(priv, fill(draft, ed25519.Sign(priv, []byte(version))))
	}
	for _, v := range []string{"0.8.0", "0.10.0", "0.9.0", "0.9.0"} {
		publish(v, "", 10)
	}
	publish("0.11.0", "beta", 10)
	publish("0.10.1", "", 3)
	content, err := verifySigned(feed, mustKey(t, pub))
	if err != nil {
		t.Fatal(err)
	}
	f, _ := parse(content)
	var got []string
	for _, it := range f.Items {
		got = append(got, it.Version+"/"+channelName(it.Channel))
	}
	if want := "0.11.0/beta 0.10.1/stable 0.10.0/stable 0.9.0/stable"; strings.Join(got, " ") != want {
		t.Errorf("items %v, want %s", got, want)
	}
}

func mustKey(t *testing.T, s string) ed25519.PublicKey {
	t.Helper()
	k, err := publicKey(s)
	if err != nil {
		t.Fatal(err)
	}
	return k
}

// verify accepts the signed feed whose release item names the disk image by its length and a
// signature over its bytes, and refuses each way it can be wrong.
func TestVerify(t *testing.T) {
	priv, pub := testKey(t)
	dir := t.TempDir()
	dmg := filepath.Join(dir, "Cogpit-0.5.0-macos-arm64.dmg")
	image := []byte("a disk image's bytes")
	if err := os.WriteFile(dmg, image, 0o600); err != nil {
		t.Fatal(err)
	}
	o := baseOptions(t, pub, "0.5.0")
	o.length = int64(len(image))
	draft, err := add(o)
	if err != nil {
		t.Fatal(err)
	}
	write := func(data []byte) string {
		p := filepath.Join(t.TempDir(), "appcast.xml")
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	good := write(signFeed(priv, fill(draft, ed25519.Sign(priv, image))))
	if items, err := verify(verifyOptions{feed: good, pubkey: pub, dmg: dmg, version: "0.5.0"}); err != nil || len(items) != 1 {
		t.Fatalf("a good feed: %v", err)
	}
	_, otherPub := testKey(t)
	for name, o := range map[string]verifyOptions{
		"another key":         {feed: good, pubkey: otherPub, dmg: dmg, version: "0.5.0"},
		"no item":             {feed: good, pubkey: pub, dmg: dmg, version: "0.6.0"},
		"draft unsigned":      {feed: write(draft), pubkey: pub},
		"image not signed":    {feed: write(signFeed(priv, fill(draft, ed25519.Sign(priv, []byte("other"))))), pubkey: pub, dmg: dmg, version: "0.5.0"},
		"placeholder signed":  {feed: write(signFeed(priv, draft)), pubkey: pub},
		"another disk image":  {feed: good, pubkey: pub, dmg: filepath.Join(dir, "x.dmg"), version: "0.5.0"},
		"public key not key":  {feed: good, pubkey: "bm90IGEga2V5"},
		"signed length wrong": {feed: write(bytes.Replace(signFeed(priv, fill(draft, ed25519.Sign(priv, image))), []byte("length: "), []byte("length: 1"), 1)), pubkey: pub},
	} {
		if _, err := verify(o); err == nil {
			t.Errorf("%s: verified", name)
		}
	}
}

// Long release notes end at a line boundary with a link to the whole notes; none at all give the
// link alone.
func TestReleaseNotesBounded(t *testing.T) {
	p := filepath.Join(t.TempDir(), "notes.md")
	long := strings.Repeat("* a change of a release with many commits\n", 2000)
	if err := os.WriteFile(p, []byte(long), 0o600); err != nil {
		t.Fatal(err)
	}
	notes, err := releaseNotes(p, "https://example.invalid/r")
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) > maxNotes+200 || !strings.HasSuffix(notes, "…\n\n[The whole release notes on GitHub](https://example.invalid/r)") {
		t.Errorf("%d bytes, ending %q", len(notes), notes[len(notes)-120:])
	}
	if none, _ := releaseNotes("", "https://example.invalid/r"); none != "[The release on GitHub](https://example.invalid/r)" {
		t.Errorf("no notes: %q", none)
	}
}
