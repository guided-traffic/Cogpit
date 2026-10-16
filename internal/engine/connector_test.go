package engine

import (
	"os"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// TestReleaseBuildsLeaveTheDemoOut: the release pipeline's make dist-app builds both binaries with
// the Makefile's DIST_TAGS, and those name the tag connector_nodemo.go is built with. A tag no file
// knows is silently ignored, and a release would ship the demo cluster (D-67).
func TestReleaseBuildsLeaveTheDemoOut(t *testing.T) {
	stub, err := os.ReadFile("connector_nodemo.go")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`(?m)^//go:build (\w+)$`).FindSubmatch(stub)
	if m == nil {
		t.Fatal("connector_nodemo.go has no single-tag build constraint")
	}
	tag := string(m[1])

	mk, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	d := regexp.MustCompile(`(?m)^DIST_TAGS\s*:?=\s*(.*)$`).FindSubmatch(mk)
	if d == nil || !slices.Contains(strings.FieldsFunc(string(d[1]), func(r rune) bool { return r == ',' || r == ' ' }), tag) {
		t.Errorf("the Makefile's DIST_TAGS do not name %q: %q", tag, d)
	}
	for _, want := range []string{
		"$(MAKE) app GO_LDFLAGS='$(DIST_LDFLAGS)' GO_TAGS='$(DIST_TAGS)'", // dist-app
		"$(GO) build -tags '$(GO_TAGS)' -ldflags '$(GO_LDFLAGS)' -o $(BIN)/cogpit ./cmd/cogpit",
		"$(GO) build -tags '$(GO_TAGS)' -ldflags '$(GO_LDFLAGS)' -o $(BIN)/cogpit-cli ./cmd/cogpit-cli",
	} {
		if !strings.Contains(string(mk), want) {
			t.Errorf("the Makefile lacks %q", want)
		}
	}
}
