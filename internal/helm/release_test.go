package helm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"

	rspb "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// encode writes a release the way Helm's Secrets driver does.
func encode(t *testing.T, rls *rspb.Release, compress bool) string {
	t.Helper()
	b, err := json.Marshal(rls)
	if err != nil {
		t.Fatal(err)
	}
	if compress {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		if _, err := w.Write(b); err != nil {
			t.Fatal(err)
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		b = buf.Bytes()
	}
	return base64.StdEncoding.EncodeToString(b)
}

func releaseSecret(payload string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1", "kind": "Secret", "type": "helm.sh/release.v1",
		"metadata": map[string]any{
			"name": "sh.helm.release.v1.web.v2", "namespace": "apps",
			"labels": map[string]any{"owner": "helm", "name": "web", "version": "2", "status": "deployed"},
		},
		"data": map[string]any{"release": base64.StdEncoding.EncodeToString([]byte(payload))},
	}}
	return u
}

func TestDecodeSecret(t *testing.T) {
	for _, compress := range []bool{true, false} {
		rls := &rspb.Release{Name: "web", Namespace: "apps", Version: 2, Manifest: "kind: Service\n"}
		got, err := DecodeSecret(releaseSecret(encode(t, rls, compress)))
		if err != nil {
			t.Fatalf("compress=%v: %v", compress, err)
		}
		if got.Name != "web" || got.Namespace != "apps" || got.Version != 2 || got.Manifest != rls.Manifest {
			t.Errorf("compress=%v: got %+v", compress, got)
		}
		if got.Labels["status"] != "deployed" {
			t.Errorf("labels not taken from the Secret: %v", got.Labels)
		}
	}
}

func TestDecodeSecretRejectsBadInput(t *testing.T) {
	noData := releaseSecret("")
	unstructured.RemoveNestedField(noData.Object, "data")
	if _, err := DecodeSecret(noData); !errors.Is(err, ErrNoPayload) {
		t.Errorf("no data: err = %v, want ErrNoPayload", err)
	}
	for name, payload := range map[string]string{
		"not base64":  "!!!",
		"not JSON":    base64.StdEncoding.EncodeToString([]byte("{")),
		"broken gzip": base64.StdEncoding.EncodeToString(append([]byte{0x1f, 0x8b, 0x08}, 0, 1, 2)),
	} {
		if _, err := DecodeSecret(releaseSecret(payload)); err == nil {
			t.Errorf("%s: no error", name)
		}
	}
}

func TestDecodePayloadBoundsDecompression(t *testing.T) {
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	zero := make([]byte, 1<<20)
	for range maxPayload>>20 + 1 {
		if _, err := w.Write(zero); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := DecodePayload([]byte(base64.StdEncoding.EncodeToString(buf.Bytes()))); err == nil {
		t.Fatal("a payload above the bound decoded")
	}
}

func FuzzDecodePayload(f *testing.F) {
	f.Add([]byte(base64.StdEncoding.EncodeToString([]byte(`{"name":"web","version":1}`))))
	f.Add([]byte("H4sI"))
	f.Fuzz(func(_ *testing.T, b []byte) {
		_, _ = DecodePayload(b)
	})
}
