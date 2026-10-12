package engine

import (
	"strings"
	"testing"

	"github.com/guided-traffic/cogpit/internal/helm"
)

func indexChart(versions ...string) helm.IndexChart {
	c := helm.IndexChart{Name: "web"}
	for _, v := range versions {
		c.Versions = append(c.Versions, helm.ChartVersion{Version: v})
	}
	return c
}

// The picker's versions: the index's order (newest first), prereleases after the others, the
// release's own version in its place when the repository does not list it; a new tab starts with
// the newest version that is not a prerelease when it is newer than the release's own.
func TestUpgradeVersionsAndDefault(t *testing.T) {
	for _, c := range []struct {
		index           []string
		own, list, pick string
	}{
		{[]string{"0.3.0", "0.2.0", "0.1.0"}, "0.1.0", "0.3.0,0.2.0,0.1.0", "0.3.0"},
		{[]string{"0.4.0-rc.1", "0.3.0", "0.2.0"}, "0.2.0", "0.3.0,0.2.0,0.4.0-rc.1", "0.3.0"},
		{[]string{"0.3.0", "0.1.0"}, "0.2.0", "0.3.0,0.2.0,0.1.0", "0.3.0"},
		{[]string{"0.3.0", "0.2.0"}, "0.0.9", "0.3.0,0.2.0,0.0.9", "0.3.0"},
		{[]string{"0.2.0", "0.1.0"}, "0.5.0", "0.5.0,0.2.0,0.1.0", "0.5.0"},                     // the repository lists only older ones
		{[]string{"1.0.0-rc.2", "1.0.0-rc.1"}, "0.9.0", "0.9.0,1.0.0-rc.2,1.0.0-rc.1", "0.9.0"}, // prereleases only: not preselected
		{[]string{"1.0.0", "0.9.0"}, "1.0.0-rc.1", "1.0.0,0.9.0,1.0.0-rc.1", "1.0.0"},
		{[]string{"2.0.0", "1.0.0"}, "custom", "2.0.0,1.0.0,custom", "custom"}, // a version that does not parse
	} {
		ic := indexChart(c.index...)
		if got := strings.Join(upgradeVersions(ic, c.own), ","); got != c.list {
			t.Errorf("versions of %v with %s: %s, want %s", c.index, c.own, got, c.list)
		}
		if got := newestUpgrade(ic, c.own); got != c.pick {
			t.Errorf("default of %v with %s: %s, want %s", c.index, c.own, got, c.pick)
		}
	}
	found := []repoChart{{repo: "a", chart: indexChart("0.3.0")}, {repo: "b", chart: indexChart("0.2.0", "0.1.0")}}
	if got := defaultUpgradeRepository(found, "0.1.0").repo; got != "b" {
		t.Errorf("the repository that lists the release's version: %s", got)
	}
	if got := defaultUpgradeRepository(found, "0.0.1").repo; got != "a" {
		t.Errorf("none lists it, the first: %s", got)
	}
}
