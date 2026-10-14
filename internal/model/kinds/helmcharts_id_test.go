package kinds

import (
	"regexp"
	"testing"
)

// A chart's id tells its repository from its chart whatever dots either name holds (A36): a dot of
// the repository's name is doubled, the first single dot separates. The id of a repository without
// a dot is what it always was, so ids Cogpit kept before read as they did.
func TestHelmChartName(t *testing.T) {
	for _, c := range []struct{ repository, chart, id string }{
		{"bitnami", "nginx", "bitnami.nginx"}, // the form before A36, unchanged
		{"bitnami", "my.chart", "bitnami.my.chart"},
		{"charts.example", "nginx", "charts..example.nginx"},
		{"charts.example", "my.chart", "charts..example.my.chart"},
		{"a.b.c", "d.e", "a..b..c.d.e"},
		{"a..b", "c", "a....b.c"},
		{"a.", "c", "a...c"},
		{"a", "c..d", "a.c..d"},
		{"prometheus-community", "kube-prometheus-stack", "prometheus-community.kube-prometheus-stack"},
		{"my_repo", "x_y", "my_repo.x_y"},
	} {
		if got := HelmChartName(c.repository, c.chart); got != c.id {
			t.Errorf("HelmChartName(%q, %q) = %q, want %q", c.repository, c.chart, got, c.id)
		}
		r, ch, ok := SplitHelmChartName(c.id)
		if !ok || r != c.repository || ch != c.chart {
			t.Errorf("SplitHelmChartName(%q) = %q, %q, %v; want %q, %q", c.id, r, ch, ok, c.repository, c.chart)
		}
		obj := HelmChartObject(HelmChart{Repository: c.repository, Name: c.chart, Version: "1.0.0"}, "uid", "1")
		if obj.GetName() != c.id {
			t.Errorf("the object of %s/%s is named %q, want %q", c.repository, c.chart, obj.GetName(), c.id)
		}
	}
	for _, bad := range []string{"", "nginx", ".nginx", "bitnami.", "a..b", "..", "...", "a..."} {
		if r, c, ok := SplitHelmChartName(bad); ok {
			t.Errorf("SplitHelmChartName(%q) = %q, %q: an id without a repository and a chart", bad, r, c)
		}
	}
}

// Every repository and chart name Cogpit lists round-trips through its id: the repository's name as
// helm.CheckRepositoryName takes it, the chart's as the index's charts are kept (helm's patterns,
// repeated here: kinds may not import helm).
var (
	idRepositoryName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,62}$`)
	idChartName      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,200}$`)
)

func FuzzHelmChartName(f *testing.F) {
	for _, s := range [][2]string{{"bitnami", "nginx"}, {"charts.example", "my.chart"}, {"a.", "b"}, {"a..b", "c..d"}, {"x", "y.z."}} {
		f.Add(s[0], s[1])
	}
	f.Fuzz(func(t *testing.T, repository, chart string) {
		if !idRepositoryName.MatchString(repository) || !idChartName.MatchString(chart) {
			return
		}
		id := HelmChartName(repository, chart)
		r, c, ok := SplitHelmChartName(id)
		if !ok || r != repository || c != chart {
			t.Fatalf("%q/%q → %q → %q, %q, %v", repository, chart, id, r, c, ok)
		}
	})
}
