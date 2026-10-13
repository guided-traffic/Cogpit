package commands_test

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// object_get gives an agent the decoded fields of every certificate a Secret holds (D-30), in its
// certificates and in the drawer's sections, and never the PEM text, a value of any key or the
// private key: the result is searched for every value of the Secret as the API sends it (base64),
// decoded, and each line of the PEM bodies; nor does the audit log, an event or the log carry one.
func TestObjectGetDecodesCertificatesForAgents(t *testing.T) {
	h := newSecretsHarness(t)
	now := time.Now()
	root := testkit.NewCert(t, testkit.CertSpec{CommonName: "Example Root CA", CA: true, NotBefore: now.AddDate(-1, 0, 0), NotAfter: now.AddDate(5, 0, 0)})
	leaf := testkit.NewCert(t, testkit.CertSpec{
		CommonName: "shop.example.com", DNSNames: []string{"shop.example.com"}, IPAddresses: []string{"10.1.2.3"},
		NotBefore: now.AddDate(0, -1, 0), NotAfter: now.Add(10*24*time.Hour + time.Hour), Parent: root,
	})
	keyPEM := leaf.KeyPEM(t)
	data := map[string]string{"tls.crt": string(testkit.Chain(leaf, root)), "tls.key": string(keyPEM), "ca.crt": string(root.PEM())}
	sec := testkit.Secret("apps", "shop-tls", nil, data)
	sec.Object["type"] = "kubernetes.io/tls"
	h.src.Set(testkit.SecretsGVR, sec)
	// A Secret whose tls.key belongs to another certificate (D-30's key check).
	wrongKey := testkit.NewCert(t, testkit.CertSpec{CommonName: "other"}).KeyPEM(t)
	wrong := testkit.Secret("apps", "shop-wrong-key", nil, map[string]string{"tls.crt": string(leaf.PEM()), "tls.key": string(wrongKey)})
	wrong.Object["type"] = "kubernetes.io/tls"
	h.src.Set(testkit.SecretsGVR, wrong)

	var forbidden []string
	for _, v := range []string{data["tls.crt"], data["tls.key"], data["ca.crt"], string(wrongKey)} {
		forbidden = append(forbidden, v, base64.StdEncoding.EncodeToString([]byte(v)))
		for _, line := range strings.Split(v, "\n") {
			if len(line) >= 16 && !strings.HasPrefix(line, "-----") {
				forbidden = append(forbidden, line)
			}
		}
	}
	forbidden = append(forbidden, "PRIVATE KEY", "BEGIN CERTIFICATE")
	carries := func(b []byte) string {
		for _, f := range forbidden {
			if strings.Contains(string(b), f) {
				return f[:min(len(f), 40)]
			}
		}
		return leak(string(b))
	}

	ref := model.ObjectRef{Context: "test", Kind: "secrets", Namespace: "apps", Name: "shop-tls"}
	for _, params := range []map[string]any{{"ref": ref}, {"ref": ref, "related": true}} {
		res, err := h.dispatch(engine.OriginExternalAgent, "object.get", params)
		if err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(res)
		if l := carries(b); l != "" {
			t.Errorf("object.get %v carries %q", params, l)
		}
		var m engine.DetailModel
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatal(err)
		}
		if len(m.Certificates) != 2 || m.Certificates[0].Key != "tls.crt" || m.Certificates[1].Key != "ca.crt" ||
			len(m.Certificates[0].Certificates) != 2 {
			t.Fatalf("certificates %+v", m.Certificates)
		}
		c := m.Certificates[0].Certificates[0]
		if c.SubjectCN != "shop.example.com" || c.IssuerCN != "Example Root CA" || c.ExpiresInDays != 10 || c.Expired ||
			c.Status != model.StatusWarning || c.NotAfter.Location() != time.UTC || strings.Join(c.IPAddresses, ",") != "10.1.2.3" {
			t.Errorf("the leaf for agents %+v", c)
		}
		for _, s := range []string{`"expiresInDays":10`, `"expired":false`, `"notAfter":"` + c.NotAfter.Format(time.RFC3339) + `"`, `"Certificates · tls.crt"`,
			`"pairedKey":"tls.key"`, `"keyMatches":true`, `"chainInOrder":true`, `"Yes (tls.key)"`} {
			if !strings.Contains(string(b), s) {
				t.Errorf("object.get %v has no %s", params, s)
			}
		}
	}
	wrongRef := model.ObjectRef{Context: "test", Kind: "secrets", Namespace: "apps", Name: "shop-wrong-key"}
	res, err := h.dispatch(engine.OriginExternalAgent, "object.get", map[string]any{"ref": wrongRef})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(res)
	if l := carries(b); l != "" {
		t.Errorf("object.get of the Secret with another key carries %q", l)
	}
	if !strings.Contains(string(b), `"keyMatches":false`) || !strings.Contains(string(b), `"No (tls.key holds another key)"`) {
		t.Errorf("object.get does not say the key belongs to another certificate: %s", b)
	}
	if l := carries([]byte(h.trail())); l != "" {
		t.Errorf("%q in the audit log, the events or the log", l)
	}
}
