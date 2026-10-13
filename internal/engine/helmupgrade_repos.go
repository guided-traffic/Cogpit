package engine

import (
	"fmt"
	"slices"

	"github.com/guided-traffic/cogpit/internal/helm"
)

// A Helm Upgrade tab's chart versions from the user's chart repositories (G11, D-29, architecture
// §8): the release's chart is looked up by name in the index of every repository of Helm's
// repositories file, as Cogpit holds it (a download that runs is not waited for). Without one the tab offers the
// release's own version only, as before; with one its picker offers that repository's versions and
// the upgrade downloads the chosen one (commands/helmupgrade.go). Cogpit's defaults, inferred, not
// the owner's decisions (Lens's choice is not captured):
//
//   - several repositories list the chart: the first, in the order of Helm's file, whose index
//     lists the release's own version (likely the one it was installed from), else the first;
//   - the version a new tab starts with: the repository's newest version that is not a
//     prerelease, when it is newer than the release's own; else the release's own;
//   - the picker: the repository's versions newest first, the prereleases after the others (as
//     helm search leaves them out unless asked), and the release's own version among them even
//     when the repository does not list it, marked as the current one;
//   - the release's own version upgrades with the release's stored chart, nothing downloaded.

// HelmUpgradePick is a choice in a Helm Upgrade tab: the repository whose versions its picker
// offers and the version the upgrade installs. An empty field keeps the tab's choice, or on a new
// tab takes the default.
type HelmUpgradePick struct {
	Repository string
	Version    string
}

// repoChart is a chart as one repository's index lists it.
type repoChart struct {
	repo  string
	chart helm.IndexChart
}

// chartIn returns the repositories whose index lists a chart, in the order of Helm's file. It does
// not wait for a download that runs. The charts' versions are shared with the indexes: read-only.
func (h *HelmRepos) chartIn(name string) ([]repoChart, error) {
	if h == nil {
		return nil, nil
	}
	if err := h.ensureLoaded(); err != nil {
		return nil, err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	var out []repoChart
	for _, e := range h.entries {
		st := h.state[e.info.Name]
		if e.repo == nil || st == nil || st.index == nil {
			continue
		}
		if c, ok := st.index.Chart(name); ok {
			out = append(out, repoChart{repo: e.info.Name, chart: c})
		}
	}
	return out, nil
}

// ChartRepositories returns the names of the repositories whose index lists a chart, at a version
// when version is not empty, in the order of Helm's file. It reads the indexes Cogpit holds and
// does not wait for a download that runs.
func (h *HelmRepos) ChartRepositories(chart, version string) ([]string, error) {
	found, err := h.chartIn(chart)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range found {
		if _, ok := f.chart.Version(version); version == "" || ok {
			out = append(out, f.repo)
		}
	}
	return out, nil
}

// IndexedVersion returns a chart version as a repository's index lists it, version "" the one the
// chart is listed with (the newest that is not a prerelease). It reads the index Cogpit holds and
// does not wait for a download that runs.
func (h *HelmRepos) IndexedVersion(repository, chart, version string) (helm.ChartVersion, error) {
	found, err := h.chartIn(chart)
	if err != nil {
		return helm.ChartVersion{}, err
	}
	i := slices.IndexFunc(found, func(f repoChart) bool { return f.repo == repository })
	if i < 0 {
		if _, err := h.repo(repository); err != nil {
			return helm.ChartVersion{}, err
		}
		return helm.ChartVersion{}, fmt.Errorf("repository %s has no chart %s (helm_repo_update downloads its index again)", repository, chart)
	}
	var v helm.ChartVersion
	var ok bool
	if version == "" {
		v, ok = found[i].chart.Latest()
	} else {
		v, ok = found[i].chart.Version(version)
	}
	if !ok {
		return helm.ChartVersion{}, fmt.Errorf("chart %s/%s has no version %s", repository, chart, version)
	}
	return v, nil
}

// upgradeChoice is what a Helm Upgrade tab on a release of chart at version own offers and has
// chosen: the repositories that list the chart, the one whose versions the picker offers, its
// versions and the version the upgrade installs, from pick, else from the tab's current choice
// cur (nil for a new tab), else the defaults above.
func (s *Session) upgradeChoice(chart, own string, pick HelmUpgradePick, cur *HelmUpgradeTab) (*HelmUpgradeTab, error) {
	t := &HelmUpgradeTab{Chart: chart, Version: own, ReleaseVersion: own, Versions: []string{own}, UserSuppliedOnly: true}
	found, err := s.opts.HelmRepos.chartIn(chart)
	if err != nil {
		if pick.Repository != "" {
			return nil, err
		}
		s.log.Warn("engine: the Helm Upgrade tab offers the release's own version only", "err", err)
		found = nil
	}
	if pick.Repository != "" && !slices.ContainsFunc(found, func(f repoChart) bool { return f.repo == pick.Repository }) {
		return nil, fmt.Errorf("no Helm repository named %s lists chart %s", pick.Repository, chart)
	}
	if len(found) == 0 {
		if pick.Version != "" && pick.Version != own {
			return nil, fmt.Errorf("chart %s %s: %w (the release's own is %s)", chart, pick.Version, helm.ErrNoRepository, own)
		}
		return t, nil
	}
	for _, f := range found {
		t.Repositories = append(t.Repositories, f.repo)
	}
	name := pick.Repository
	keep := cur != nil && cur.Repository != "" && slices.Contains(t.Repositories, cur.Repository)
	if name == "" && keep {
		name = cur.Repository
	}
	var rc repoChart
	if name == "" {
		rc = defaultUpgradeRepository(found, own)
	} else {
		rc = found[slices.IndexFunc(found, func(f repoChart) bool { return f.repo == name })]
	}
	t.Repository = rc.repo
	t.Versions = upgradeVersions(rc.chart, own)
	switch {
	case pick.Version != "":
		if !slices.Contains(t.Versions, pick.Version) {
			return nil, fmt.Errorf("chart %s/%s has no version %s", rc.repo, chart, pick.Version)
		}
		t.Version = pick.Version
	case pick.Repository == "" && keep && cur.Repository == rc.repo && slices.Contains(t.Versions, cur.Version):
		t.Version = cur.Version
	default:
		t.Version = newestUpgrade(rc.chart, own)
	}
	return t, nil
}

// defaultUpgradeRepository picks the repository a new tab offers: the first whose index lists the
// release's own version, else the first.
func defaultUpgradeRepository(found []repoChart, own string) repoChart {
	for _, f := range found {
		if _, ok := f.chart.Version(own); ok {
			return f
		}
	}
	return found[0]
}

// upgradeVersions are a picker's choices: the repository's versions newest first (the index's
// order), the prereleases after the others, and the release's own version among them, in its place
// by version, when the repository does not list it.
func upgradeVersions(c helm.IndexChart, own string) []string {
	var stable, pre []string
	for _, v := range c.Versions {
		if v.Prerelease() {
			pre = append(pre, v.Version)
		} else {
			stable = append(stable, v.Version)
		}
	}
	if !slices.Contains(stable, own) && !slices.Contains(pre, own) {
		if (helm.ChartVersion{Version: own}).Prerelease() {
			pre = insertVersion(pre, own)
		} else {
			stable = insertVersion(stable, own)
		}
	}
	return append(stable, pre...)
}

// insertVersion inserts v into versions (newest first) before the first older one, or at the end.
func insertVersion(versions []string, v string) []string {
	i := slices.IndexFunc(versions, func(x string) bool {
		c, ok := helm.CompareVersions(x, v)
		return ok && c < 0
	})
	if i < 0 {
		return append(versions, v)
	}
	return slices.Insert(versions, i, v)
}

// newestUpgrade is the version a new tab starts with: the chart's newest version that is not a
// prerelease when it is newer than the release's own, else the release's own.
func newestUpgrade(c helm.IndexChart, own string) string {
	latest, ok := c.Latest()
	if !ok || latest.Prerelease() {
		return own
	}
	if cmp, ok := helm.CompareVersions(latest.Version, own); ok && cmp > 0 {
		return latest.Version
	}
	return own
}

// setHelmUpgradeChoice gives an open Helm Upgrade tab a new choice, in the dock and then in its
// editor, whose EventEditor (the text's revision unchanged) tells the GUI's tab to read the dock
// again: a tab's choice is not among what makes the GUI sync its dock.
func (s *Session) setHelmUpgradeChoice(id string, t *HelmUpgradeTab) error {
	if _, err := s.editorOf(id); err != nil {
		return err
	}
	s.UpdateUI(func(u *UIState) {
		for i, x := range u.Dock {
			if x.ID == id {
				c := *t
				u.Dock[i].Helm = &c
			}
		}
	})
	_, err := s.updateEditor(id, func(e *editor) {
		c := *t // a new value: copies of the editor share the old one
		e.tab.Helm = &c
	})
	return err
}
