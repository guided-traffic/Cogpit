package fyneui

import "testing"

// A source context's id carries its file ("<context> [<file>]", D-28); the bars show the name and
// the file apart, so the sidebar's header and the title bar show the name, and the file muted.
func TestContextNameAndFileShownApart(t *testing.T) {
	for id, want := range map[string][2]string{
		"awe-d-pinniped [.kube/kubeconfig-awe-d.yaml]": {"awe-d-pinniped", ".kube/kubeconfig-awe-d.yaml"},
		"demo":                   {"demo", ""},
		"prod [a.yaml]":          {"prod", "a.yaml"},
		"odd [name] [b/c.yaml]":  {"odd [name]", "b/c.yaml"},
		"no-file-but-bracket [x": {"no-file-but-bracket [x", ""},
	} {
		if n, f := splitContextID(id); n != want[0] || f != want[1] {
			t.Errorf("splitContextID(%q) = %q, %q", id, n, f)
		}
	}

	f := connectedFake(t, 10)
	f.ui.Context = "awe-d-pinniped [.kube/kubeconfig-awe-d.yaml]"
	sh, _, q := newTestShell(t, f, Options{Theme: ThemeDark})
	q.until(t, "the header", func() bool { return sh.side.name.text == "awe-d-pinniped" })
	if sh.side.badge == nil || sh.top.context.text != "awe-d-pinniped" || sh.top.file.text != ".kube/kubeconfig-awe-d.yaml" || !sh.top.file.Visible() {
		t.Errorf("title bar %q / %q (visible %v)", sh.top.context.text, sh.top.file.text, sh.top.file.Visible())
	}
}
