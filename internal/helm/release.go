package helm

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	rspb "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// ReleaseSelector is the label selector of the records Helm keeps its releases in: one Secret
// (the default storage driver) or ConfigMap (HELM_DRIVER=configmap, configmaps.go) per release
// revision, named sh.helm.release.v1.<release>.v<revision>.
const ReleaseSelector = "owner=helm"

// maxPayload bounds a release payload after decompression: a release Secret is cluster data and
// is decoded as untrusted input (a gzip bomb must not exhaust memory). Kubernetes caps a Secret at
// 1 MiB, compressed; Helm's own payloads stay far below this bound.
const maxPayload = 64 << 20

// magicGzip starts a gzip stream; Helm compresses every payload it writes since Helm 2.
var magicGzip = []byte{0x1f, 0x8b, 0x08}

// ErrNoPayload marks a release record (a Secret or a ConfigMap) without a release payload
// (data.release missing or empty): one held as metadata only, or not Helm's.
var ErrNoPayload = errors.New("the record holds no Helm release payload")

// DecodeSecret decodes the release a Helm release Secret holds: data.release is the Kubernetes
// base64 of Helm's own encoding (base64 of gzip of JSON, DecodePayload). The release's Labels
// come from the Secret's labels, as Helm's driver sets them.
func DecodeSecret(secret *unstructured.Unstructured) (*rspb.Release, error) {
	data, _, _ := unstructured.NestedString(secret.Object, "data", "release")
	if data == "" {
		return nil, ErrNoPayload
	}
	raw, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("release Secret %s/%s: %w", secret.GetNamespace(), secret.GetName(), err)
	}
	rls, err := DecodePayload(raw)
	if err != nil {
		return nil, fmt.Errorf("release Secret %s/%s: %w", secret.GetNamespace(), secret.GetName(), err)
	}
	rls.Labels = secret.GetLabels()
	return rls, nil
}

// DecodePayload decodes Helm's release encoding: base64 of a gzip stream (or, for releases stored
// before Helm compressed them, of the plain JSON) of the release.
func DecodePayload(payload []byte) (*rspb.Release, error) {
	b := make([]byte, base64.StdEncoding.DecodedLen(len(payload)))
	n, err := base64.StdEncoding.Decode(b, payload)
	if err != nil {
		return nil, fmt.Errorf("decode the release payload: %w", err)
	}
	b = b[:n]
	if len(b) > 3 && bytes.Equal(b[:3], magicGzip) {
		zr, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, fmt.Errorf("decompress the release payload: %w", err)
		}
		defer zr.Close()
		plain, err := io.ReadAll(io.LimitReader(zr, maxPayload+1))
		if err != nil {
			return nil, fmt.Errorf("decompress the release payload: %w", err)
		}
		if len(plain) > maxPayload {
			return nil, fmt.Errorf("the release payload is larger than %s", model.FormatBytes(maxPayload))
		}
		b = plain
	}
	var rls rspb.Release
	if err := json.Unmarshal(b, &rls); err != nil {
		return nil, fmt.Errorf("parse the release payload: %w", err)
	}
	return &rls, nil
}
