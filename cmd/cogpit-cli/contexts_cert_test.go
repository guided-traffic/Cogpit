package main

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// CLIENT CERT says how long a context's client certificate is valid, as the app's cluster list
// does (D-75): "expires in 23 days", "expired 3 days ago"; empty for a token. The certificate and
// the key are not printed.
func TestContextsShowTheClientCertificate(t *testing.T) {
	te := newTestEnv(t)
	now := time.Now()
	valid := testkit.NewCert(t, testkit.CertSpec{NotBefore: now.Add(-time.Hour), NotAfter: now.Add(23*24*time.Hour + time.Hour)})
	expired := testkit.NewCert(t, testkit.CertSpec{NotBefore: now.Add(-90 * 24 * time.Hour), NotAfter: now.Add(-3*24*time.Hour - time.Hour)})
	b64 := func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
	te.writeKube(t, "config", `apiVersion: v1
kind: Config
clusters:
- name: c
  cluster: {server: "https://c.example:6443"}
contexts:
- {name: valid, context: {cluster: c, user: valid}}
- {name: expired, context: {cluster: c, user: expired}}
- {name: token, context: {cluster: c, user: token}}
users:
- name: valid
  user: {client-certificate-data: `+b64(valid.PEM())+`, client-key-data: `+b64(valid.KeyPEM(t))+`}
- name: expired
  user: {client-certificate-data: `+b64(expired.PEM())+`, client-key-data: `+b64(expired.KeyPEM(t))+`}
- name: token
  user: {token: tok-SECRET}
`)
	if code := te.run(t, "contexts"); code != exitOK {
		t.Fatalf("exit code %d; stderr: %s", code, te.errOut)
	}
	out := te.out.String()
	rows := map[string]string{}
	for _, l := range strings.Split(strings.TrimSpace(out), "\n") {
		rows[strings.Fields(l)[0]] = l
	}
	if h := rows["CONTEXT"]; !strings.HasSuffix(h, "CLIENT CERT") {
		t.Errorf("header %q", h)
	}
	for name, want := range map[string]string{"valid": "expires in 23 days", "expired": "expired 3 days ago"} {
		if l := rows[name]; !strings.HasSuffix(l, want) {
			t.Errorf("%s's row %q, want it to end in %q", name, l, want)
		}
	}
	if l := rows["token"]; strings.Contains(l, "expire") {
		t.Errorf("the token's row %q", l)
	}
	for _, leak := range []string{"CERTIFICATE-----", "PRIVATE KEY", b64(valid.PEM())[:40], "tok-SECRET"} {
		if strings.Contains(out, leak) {
			t.Errorf("the table holds %q", leak)
		}
	}
}
