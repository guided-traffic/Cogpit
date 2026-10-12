package fyneui

import (
	"context"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/kubeconfig"
)

// overlayTexts returns the text of every label, text line and paragraph in the window's top
// overlay (a dialog), in tree order.
func overlayTexts(w fyne.Window) []string {
	overlay := w.Canvas().Overlays().Top()
	if overlay == nil {
		return nil
	}
	var out []string
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || !o.Visible() {
			return
		}
		switch x := o.(type) {
		case *textLine:
			out = append(out, x.text)
		case *paragraph:
			out = append(out, x.text)
		case *widget.Label:
			out = append(out, x.Text)
		case *fyne.Container:
			for _, c := range x.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range test.WidgetRenderer(x).Objects() {
				walk(c)
			}
		}
	}
	walk(overlay)
	return out
}

func hasText(texts []string, sub string) bool {
	for _, t := range texts {
		if strings.Contains(t, sub) {
			return true
		}
	}
	return false
}

// approvalFake is the kubeconfig fake with a plugin description that tweak changes.
func approvalFake(t *testing.T, state string, tweak func(view map[string]any, plugin map[string]any)) *fakeEngine {
	f, _ := kubeconfigFake(t, state)
	f.handlers["kubeconfig.execPlugin"] = func(context.Context, engine.Invocation) (any, error) {
		v := pluginView(state)
		tweak(v, v["plugin"].(map[string]any))
		return v, nil
	}
	return f
}

func openApproval(t *testing.T, f *fakeEngine, mode ThemeMode) (*shell, fyne.Window) {
	t.Helper()
	sh, w, q := newTestShell(t, f, Options{Theme: mode})
	selectAndConnect(t, sh, q, "prod-eks [eks.yaml]")
	q.until(t, "the approval dialog", func() bool { return sh.kc.approval != nil })
	return sh, w
}

// untrusted tweaks the EKS plugin as R-11 describes: the user's real server URL, a proxy of a
// third party, certificate checks off and a CA of the kubeconfig's own.
func untrusted(_ map[string]any, plugin map[string]any) {
	plugin["tls"] = map[string]any{
		"proxyURL": "http://proxy.example:3128", "insecure": true, "serverName": "sni.example",
		"caEmbedded": true, "caFile": "/Users/me/Downloads/ca.pem",
	}
	plugin["files"] = map[string]any{"caFileSHA256": strings.Repeat("ab", 32)}
}

// TestApprovalDialogShowsProxyAndInsecure (review R-11): the dialog shows how the credential
// travels, not only the server URL: an insecure, proxied or own-CA connection as warning lines.
func TestApprovalDialogShowsProxyAndInsecure(t *testing.T) {
	sh, w := openApproval(t, approvalFake(t, engine.ExecRequired, untrusted), ThemeDark)
	texts := overlayTexts(w)
	for _, want := range []string{
		"proxy.example:3128", "Certificate checks are off", "own certificate authority", "sni.example",
		"/Users/me/Downloads/ca.pem", "abababababababab…", "can read the login credential",
	} {
		if !hasText(texts, want) {
			t.Errorf("the dialog does not show %q; it shows %q", want, texts)
		}
	}
	// The warnings are the warning colour, not plain rows.
	warned := 0
	for _, o := range walkParagraphs(w) {
		if o.color == colorStatusWarning && (strings.Contains(o.text, "Certificate checks are off") ||
			strings.Contains(o.text, "proxy.example") || strings.Contains(o.text, "own certificate authority")) {
			warned++
		}
	}
	if warned < 3 {
		t.Errorf("%d of 3 warning lines are in the warning colour", warned)
	}
	_ = sh
}

// walkParagraphs returns the paragraphs of the top overlay.
func walkParagraphs(w fyne.Window) []*paragraph {
	var out []*paragraph
	var walk func(o fyne.CanvasObject)
	walk = func(o fyne.CanvasObject) {
		if o == nil || !o.Visible() {
			return
		}
		switch x := o.(type) {
		case *paragraph:
			out = append(out, x)
		case *fyne.Container:
			for _, c := range x.Objects {
				walk(c)
			}
		case fyne.Widget:
			for _, c := range test.WidgetRenderer(x).Objects() {
				walk(c)
			}
		}
	}
	walk(w.Canvas().Overlays().Top())
	return out
}

// TestApprovalDialogPlainTLSAddsNothing: a connection with the system's trust and no proxy shows no
// extra lines.
func TestApprovalDialogPlainTLSAddsNothing(t *testing.T) {
	_, w := openApproval(t, approvalFake(t, engine.ExecRequired, func(map[string]any, map[string]any) {}), ThemeDark)
	texts := overlayTexts(w)
	for _, not := range []string{"Certificate check", "Proxy", "Trusted CA", "TLS server name", "certificate authority"} {
		if hasText(texts, not) {
			t.Errorf("a plain connection shows %q: %q", not, texts)
		}
	}
}

// TestApprovalDialogEscapesUntrustedText: proxy URL, CA path and token file path are kubeconfig
// text and go through DisplayText.
func TestApprovalDialogEscapesUntrustedText(t *testing.T) {
	f := approvalFake(t, engine.ExecRequired, func(_ map[string]any, plugin map[string]any) {
		plugin["tls"] = map[string]any{"proxyURL": "http://pro\u202exy.example:3128", "caFile": "/ca\n.pem", "serverName": "a\u200bb"}
	})
	_, w := openApproval(t, f, ThemeDark)
	texts := overlayTexts(w)
	for _, t2 := range texts {
		if strings.ContainsAny(t2, "\u202e\u200b\n") {
			t.Errorf("unescaped control characters in %q", t2)
		}
	}
	if !hasText(texts, "pro"+`\u202e`+"xy.example") || !hasText(texts, `/ca\u000a.pem`) {
		t.Errorf("the escapes are visible: %q", texts)
	}
}

// TestApprovalDialogShowsProgramDigest: a program inside the kubeconfig's folder is shown with the
// digest the approval binds.
func TestApprovalDialogShowsProgramDigest(t *testing.T) {
	f := approvalFake(t, engine.ExecRequired, func(_ map[string]any, plugin map[string]any) {
		plugin["command"] = "/Users/me/Downloads/plug"
		plugin["files"] = map[string]any{"programSHA256": strings.Repeat("cd", 32)}
	})
	_, w := openApproval(t, f, ThemeDark)
	texts := overlayTexts(w)
	if !hasText(texts, "inside the kubeconfig's folder; content SHA-256 cdcdcdcdcdcdcdcd…") || !hasText(texts, "replacing the file asks again") {
		t.Errorf("%q", texts)
	}
}

// tokenFileTweak makes the plugin a token file alone, as D-17 asks for.
func tokenFileTweak(view map[string]any, plugin map[string]any) {
	view["kind"] = kubeconfig.LoginTokenFile
	delete(plugin, "command")
	delete(plugin, "args")
	delete(plugin, "envNames")
	plugin["tokenFile"] = "/Users/me/secrets/prod.token"
	view["found"] = ""
	view["riskyEnv"] = nil
}

// TestApprovalDialogAsksForATokenFile (D-17): the same dialog names the file and the server its
// content goes to; there is no program to show.
func TestApprovalDialogAsksForATokenFile(t *testing.T) {
	f := approvalFake(t, engine.ExecRequired, tokenFileTweak)
	sh, w := openApproval(t, f, ThemeDark)
	texts := overlayTexts(w)
	for _, want := range []string{
		"Allow this file as the login token?", "/Users/me/secrets/prod.token", "ABCDEF.gr7.eu-central-1.eks.amazonaws.com",
		"Token file", "Credential for",
	} {
		if !hasText(texts, want) {
			t.Errorf("the dialog does not show %q; it shows %q", want, texts)
		}
	}
	for _, not := range []string{"Program", "Arguments", "Environment"} {
		if hasText(texts, not) {
			t.Errorf("a token file has no %q: %q", not, texts)
		}
	}
	_ = sh
}

func TestRenderExecApprovalUntrustedDark(t *testing.T) {
	_, w := openApproval(t, approvalFake(t, engine.ExecRequired, untrusted), ThemeDark)
	savePNG(t, w, "exec_approval_untrusted_dark.png")
}

func TestRenderTokenFileApprovalLight(t *testing.T) {
	_, w := openApproval(t, approvalFake(t, engine.ExecRequired, tokenFileTweak), ThemeLight)
	savePNG(t, w, "token_file_approval_light.png")
}
