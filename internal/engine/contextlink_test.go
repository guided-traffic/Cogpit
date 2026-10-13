package engine

import "testing"

// ofSource tells a source's contexts by their ids' labels: a file source's id is the whole label,
// a folder's id is followed by the name of one of its files; a context of another source whose
// name or id looks alike is not the source's.
func TestOfSource(t *testing.T) {
	file := KubeconfigSource{ID: "team.yaml", Kind: SourceFile}
	folder := KubeconfigSource{ID: "synced/", Kind: SourceFolder}
	for _, tt := range []struct {
		src  KubeconfigSource
		id   string
		want bool
	}{
		{file, "team-a [team.yaml]", true},
		{file, "x [y] [team.yaml]", true},
		{file, "team-a [team.yaml (2)]", false},
		{file, "team-a [old-team.yaml]", false},
		{file, "team-a [synced/team.yaml]", false},
		{file, "team.yaml", false},
		{file, "demo", false},
		{folder, "edge [synced/edge.yaml]", true},
		{folder, "edge [synced/]", false},
		{folder, "edge [synced (2)/edge.yaml]", false},
		{folder, "edge [not-synced/edge.yaml]", false},
		{folder, "edge [synced/sub/edge.yaml]", false},
		{KubeconfigSource{Kind: SourceFile}, "edge []", false},
	} {
		if got := ofSource(tt.src, tt.id); got != tt.want {
			t.Errorf("ofSource(%q, %q) = %v, want %v", tt.src.ID, tt.id, got, tt.want)
		}
	}
}
