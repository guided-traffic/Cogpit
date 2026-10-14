package fyneui

import (
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Create Secret (G9, D-19) against the fake engine: the Secrets view's "+" opens the dialog, a value
// is masked until the eye shows it and may span lines, a type fills in its keys without touching
// what was typed, and the parameters the dialog sends are the ones the real registry takes. PNGs:
// testdata/create_secret_dialog_{dark,light}.png.

const testCertificate = "-----BEGIN CERTIFICATE-----\nMIIBszCCAVmgAwIBAgIUQ\n-----END CERTIFICATE-----\n"

// setRow types texts into a row's fields.
func setRow(r *editorRow, texts ...string) {
	for i, txt := range texts {
		r.fields[i].(interface{ SetText(string) }).SetText(txt)
	}
}

// valueEntry is a data row's value field.
func valueEntry(r *editorRow) *widget.Entry { return &r.fields[1].(*secretEntry).Entry }

// rowKeys returns the keys of an editor's rows, empty ones included.
func rowKeys(e *rowEditor) []string {
	out := make([]string, 0, len(e.rows))
	for _, r := range e.rows {
		out = append(out, rowKey(r).Text)
	}
	return out
}

// TestSecretDialogSendsTheSecret: the dialog starts as an Opaque Secret in the filter's namespace,
// its value fields masked and multi-line; the TLS type fills in tls.crt and tls.key; what is sent
// is what the registry's secret.create takes, the values as typed, empty rows left out.
func TestSecretDialogSendsTheSecret(t *testing.T) {
	f := createFake(t, kindSecrets)
	sh, _, q := newCreateShell(t, f, ThemeDark)
	form := openForm(t, sh, q, kindSecrets)
	sf := form.secret
	if sf == nil || sf.typ.Selected != engine.SecretTypeOpaque || form.namespace() != "ci" {
		t.Fatalf("secret form %+v, namespace %q", sf, form.namespace())
	}
	value := valueEntry(sf.data.rows[0])
	if !value.Password || !value.MultiLine {
		t.Errorf("value field: password %v, multi-line %v", value.Password, value.MultiLine)
	}
	form.name.SetText("web-tls")
	sf.typ.SetSelected(engine.SecretTypeTLS)
	if got := rowKeys(sf.data); !slices.Equal(got, []string{"tls.crt", "tls.key"}) {
		t.Fatalf("TLS rows %q", got)
	}
	setRow(sf.data.rows[0], "tls.crt", testCertificate)
	setRow(sf.data.rows[1], "tls.key", "key-material")
	setRow(sf.labels.rows[0], "app.kubernetes.io/name", "web")
	sf.annotations.add() // empty rows are left out
	test.Tap(form.create)
	q.until(t, "the create", func() bool { return len(f.callsOf(engine.SecretCreateCommand)) == 1 && !form.shown })
	raw := sent(t, f, realRegistry(t), engine.SecretCreateCommand)
	var got map[string]any
	if err := json.Unmarshal([]byte(raw), &got); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{"name": "web-tls", "namespace": "ci", "type": engine.SecretTypeTLS,
		"data":   []any{map[string]any{"key": "tls.crt", "value": testCertificate}, map[string]any{"key": "tls.key", "value": "key-material"}},
		"labels": []any{map[string]any{"key": "app.kubernetes.io/name", "value": "web"}}, "annotations": []any{}}
	wantRaw, _ := json.Marshal(want)
	gotRaw, _ := json.Marshal(got)
	if string(gotRaw) != string(wantRaw) {
		t.Errorf("sent %s\nwant %s", gotRaw, wantRaw)
	}
}

// TestSecretDialogTypeFillsItsKeys: a type's keys go into empty rows or new ones, never over a
// typed key; a service account token gets its annotation; switching back leaves the rows alone.
func TestSecretDialogTypeFillsItsKeys(t *testing.T) {
	f := createFake(t, kindSecrets)
	sh, _, q := newCreateShell(t, f, ThemeDark)
	form := openForm(t, sh, q, kindSecrets)
	sf := form.secret
	setRow(sf.data.rows[0], "username", "alice")
	sf.typ.SetSelected(engine.SecretTypeBasicAuth)
	if got := rowKeys(sf.data); !slices.Equal(got, []string{"username", "password"}) {
		t.Errorf("basic auth rows %q", got)
	}
	if v := valueEntry(sf.data.rows[0]).Text; v != "alice" {
		t.Errorf("the typed value became %q", v)
	}
	sf.typ.SetSelected(engine.SecretTypeServiceAccountToken)
	if got := rowKeys(sf.annotations); !slices.Equal(got, []string{engine.ServiceAccountNameAnnotation}) {
		t.Errorf("token annotations %q", got)
	}
	sf.typ.SetSelected(engine.SecretTypeOpaque)
	if got := rowKeys(sf.data); !slices.Equal(got, []string{"username", "password"}) {
		t.Errorf("rows after Opaque %q", got)
	}
}

// TestSecretDialogRevealsOneValue: the eye at a value's end shows that value only, and hides it
// again.
func TestSecretDialogRevealsOneValue(t *testing.T) {
	f := createFake(t, kindSecrets)
	sh, _, q := newCreateShell(t, f, ThemeDark)
	form := openForm(t, sh, q, kindSecrets)
	sf := form.secret
	sf.typ.SetSelected(engine.SecretTypeBasicAuth)
	user, pass := valueEntry(sf.data.rows[0]), valueEntry(sf.data.rows[1])
	eye, ok := pass.ActionItem.(fyne.Tappable)
	if !ok {
		t.Fatalf("the value field has no eye: %T", pass.ActionItem)
	}
	eye.Tapped(&fyne.PointEvent{})
	if pass.Password || !user.Password {
		t.Errorf("after the eye: password shown %v, username shown %v", !pass.Password, !user.Password)
	}
	eye.Tapped(&fyne.PointEvent{})
	if !pass.Password {
		t.Error("the eye did not hide the value again")
	}
}

// fillSecret fills the dialog as the renders show it: a TLS Secret, the certificate shown, the key
// masked, a label.
func fillSecret(form *createForm, q *mainQueue) {
	form.name.SetText("web-tls")
	sf := form.secret
	sf.typ.SetSelected(engine.SecretTypeTLS)
	setRow(sf.data.rows[0], "tls.crt", strings.TrimSuffix(testCertificate, "\n"))
	setRow(sf.data.rows[1], "tls.key", "-----BEGIN PRIVATE KEY-----")
	if eye, ok := valueEntry(sf.data.rows[0]).ActionItem.(fyne.Tappable); ok {
		eye.Tapped(&fyne.PointEvent{})
	}
	setRow(sf.labels.rows[0], "app.kubernetes.io/name", "web")
	q.drain()
}

func TestRenderCreateSecretDialogDark(t *testing.T) {
	renderCreateDialog(t, kindSecrets, ThemeDark, "create_secret_dialog_dark.png", fillSecret)
}

func TestRenderCreateSecretDialogLight(t *testing.T) {
	renderCreateDialog(t, kindSecrets, ThemeLight, "create_secret_dialog_light.png", fillSecret)
}

// TestSecretDialogCopiesAShownValueConcealed (H-70): a masked value copies nothing; once the eye
// shows it, a copy goes through the shell's clipboard for Secret values (cleared after the TTL if
// the clipboard still holds it, and said so in a toast that never shows the value), and a cut also
// removes the selection from the field. The field's menu calls the same TypedShortcut (Fyne's
// Entry.TappedSecondary), so ⌘C, ⌘X and the menu take this one path.
func TestSecretDialogCopiesAShownValueConcealed(t *testing.T) {
	f := createFake(t, kindSecrets)
	sh, _, q := newCreateShell(t, f, ThemeDark)
	sh.secretClipTTL = 40 * time.Millisecond
	form := openForm(t, sh, q, kindSecrets)
	row := form.secret.data.rows[0]
	setRow(row, "password", "correct-horse")
	field := row.fields[1].(*secretEntry)
	clip := sh.app.Clipboard()

	clip.SetContent("before")
	field.TypedShortcut(&fyne.ShortcutSelectAll{})
	field.TypedShortcut(&fyne.ShortcutCopy{Clipboard: clip})
	if got := clip.Content(); got != "before" {
		t.Fatalf("a masked value was copied: the clipboard holds %q", got)
	}

	field.ActionItem.(fyne.Tappable).Tapped(&fyne.PointEvent{}) // the eye
	field.TypedShortcut(&fyne.ShortcutSelectAll{})
	field.TypedShortcut(&fyne.ShortcutCopy{Clipboard: clip})
	if got := clip.Content(); got != "correct-horse" || sh.clipSecret != "correct-horse" {
		t.Fatalf("the clipboard holds %q (kept for clearing: %q), want the shown value", got, sh.clipSecret)
	}
	var toast string
	for _, tl := range findAll[*paragraph](sh.toasts) {
		toast += tl.text
	}
	if !strings.Contains(toast, "cleared") || strings.Contains(toast, "correct-horse") {
		t.Errorf("the toast %q should say the clipboard is cleared, and never show the value", toast)
	}
	q.until(t, "the clipboard to be cleared", func() bool { return clip.Content() == "" })

	field.TypedShortcut(&fyne.ShortcutSelectAll{})
	field.TypedShortcut(&fyne.ShortcutCut{Clipboard: clip})
	if got := clip.Content(); got != "correct-horse" || field.Text != "" {
		t.Errorf("after a cut the clipboard holds %q and the field %q, want the value and an empty field", got, field.Text)
	}
}
