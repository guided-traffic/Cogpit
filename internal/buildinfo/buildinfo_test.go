package buildinfo

import (
	"os"
	"reflect"
	"regexp"
	"runtime/debug"
	"testing"
)

// buildInfo is Go's build information with a module version and VCS settings.
func buildInfo(mod string, settings ...string) *debug.BuildInfo {
	bi := &debug.BuildInfo{Main: debug.Module{Path: "github.com/guided-traffic/cogpit", Version: mod}}
	for i := 0; i+1 < len(settings); i += 2 {
		bi.Settings = append(bi.Settings, debug.BuildSetting{Key: settings[i], Value: settings[i+1]})
	}
	return bi
}

// The release's version comes from the linker; without it, a release tag Go recorded counts; every
// other build is a development build with the commit Go recorded.
func TestVersionFromTheBuild(t *testing.T) {
	const rev = "0691fb32abcdef0123456789abcdef0123456789"
	for _, tt := range []struct {
		name   string
		linked string
		bi     *debug.BuildInfo
		want   Info
		str    string
	}{
		{"the release pipeline", "0.7.0", buildInfo("v0.7.0", "vcs.revision", rev, "vcs.modified", "false"),
			Info{Version: "0.7.0", Release: true, Commit: "0691fb32abcd"}, "0.7.0"},
		{"the linker wins over the module", "0.0.99", buildInfo("v0.6.1-0.20261014150350-e922515f239b", "vcs.revision", rev),
			Info{Version: "0.0.99", Release: true, Commit: "0691fb32abcd"}, "0.0.99"},
		{"go install at a tag", "", buildInfo("v0.7.0"), Info{Version: "0.7.0", Release: true}, "0.7.0"},
		{"make app after a tag", "", buildInfo("v0.6.1-0.20261014150350-e922515f239b", "vcs.revision", rev, "vcs.modified", "false"),
			Info{Version: Dev, Commit: "0691fb32abcd"}, "dev 0691fb32abcd"},
		{"changes not committed", "", buildInfo("v0.7.0+dirty", "vcs.revision", rev, "vcs.modified", "true"),
			Info{Version: Dev, Commit: "0691fb32abcd", Modified: true}, "dev 0691fb32abcd-dirty"},
		{"a tag with changes but no mark", "", buildInfo("v0.7.0", "vcs.revision", rev, "vcs.modified", "true"),
			Info{Version: Dev, Commit: "0691fb32abcd", Modified: true}, "dev 0691fb32abcd-dirty"},
		{"a prerelease tag", "", buildInfo("v0.8.0-beta.1"), Info{Version: Dev}, "dev"},
		{"go run", "", buildInfo("(devel)"), Info{Version: Dev}, "dev"},
		{"no build information", "", nil, Info{Version: Dev}, "dev"},
		{"no build information, linked", "0.7.0", nil, Info{Version: "0.7.0", Release: true}, "0.7.0"},
	} {
		got := from(tt.linked, tt.bi)
		if got != tt.want {
			t.Errorf("%s: %+v, want %+v", tt.name, got, tt.want)
		}
		if s := got.String(); s != tt.str {
			t.Errorf("%s: String %q, want %q", tt.name, s, tt.str)
		}
		if got.Release && got.CommitText() != "" {
			t.Errorf("%s: a release shows the commit %q", tt.name, got.CommitText())
		}
	}
}

// The test binary is a development build: go test records no release.
func TestReadInATest(t *testing.T) {
	if info := Read(); info.Release || info.Version != Dev || Version() != Dev {
		t.Errorf("a test binary: %+v", info)
	}
}

// The Makefile's release flags set this package's variable: a -X naming a symbol that does not
// exist is silently ignored, and a release would say "dev".
func TestMakefileSetsTheVersion(t *testing.T) {
	b, err := os.ReadFile("../../Makefile")
	if err != nil {
		t.Fatal(err)
	}
	flag := "-X " + reflect.TypeOf(Info{}).PkgPath() + ".version=$(VERSION)"
	m := regexp.MustCompile(`(?m)^DIST_LDFLAGS\s*=\s*(.*)$`).FindSubmatch(b)
	if m == nil || !regexp.MustCompile(`(^|\s)`+regexp.QuoteMeta(flag)+`(\s|$)`).Match(m[1]) {
		t.Errorf("the Makefile's DIST_LDFLAGS do not contain %q: %q", flag, m)
	}
	_ = version // the variable the flag names
}
