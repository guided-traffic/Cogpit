package helm

import (
	"errors"
	"fmt"
	"strings"

	rspb "helm.sh/helm/v4/pkg/release/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Helm's storage drivers (G11, architecture §8). Helm keeps one record per release revision in the
// release's namespace: a Secret by default, a ConfigMap with HELM_DRIVER=configmap, both named
// sh.helm.release.v1.<release>.v<revision> and labelled owner=helm, name, status and version.
// Cogpit reads both (the Releases view, the drawer, the Upgrade tab's values) and runs every
// action on an existing release with the driver its records are in; a new install uses the
// HELM_DRIVER preference. The SQL driver keeps releases outside the cluster and is not supported.

// Driver is a Helm storage driver, as HELM_DRIVER names it.
type Driver string

// The storage drivers Cogpit reads and writes.
const (
	// DriverSecret keeps a release's records in Secrets, Helm's default.
	DriverSecret Driver = "secret"
	// DriverConfigMap keeps them in ConfigMaps (HELM_DRIVER=configmap). Their values are no less
	// sensitive: a record holds the release's values whole, in Helm's encoding, readable by anyone
	// who may get ConfigMaps in the namespace.
	DriverConfigMap Driver = "configmap"
)

// ErrSQLDriver refuses Helm's SQL storage driver: its releases live in a database outside the
// cluster, which Cogpit neither reads nor writes.
var ErrSQLDriver = errors.New("the sql storage driver of Helm keeps releases in a database outside the cluster, which Cogpit does not read or write; choose secret or configmap")

// ParseDriver reads a HELM_DRIVER value: "" or "secret" (also "secrets") is DriverSecret,
// "configmap" (also "configmaps") DriverConfigMap; "sql" is ErrSQLDriver, anything else an error.
func ParseDriver(s string) (Driver, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "secret", resourceSecrets:
		return DriverSecret, nil
	case "configmap", "configmaps":
		return DriverConfigMap, nil
	case "sql":
		return "", ErrSQLDriver
	}
	return "", fmt.Errorf("helm storage driver %q: secret or configmap", s)
}

// Resource is the API resource the driver keeps records in: "secrets" or "configmaps".
func (d Driver) Resource() string {
	if d == DriverConfigMap {
		return "configmaps"
	}
	return resourceSecrets
}

// resourceSecrets is the resource of Helm's default driver's records, and of the releases' scope.
const resourceSecrets = "secrets"

// Kind is the kind of the driver's records: "Secret" or "ConfigMap".
func (d Driver) Kind() string {
	if d == DriverConfigMap {
		return kindConfigMap
	}
	return kindSecret
}

// sdkName is the driver's name for the Helm SDK's action.Configuration.Init.
func (d Driver) sdkName() string {
	if d == DriverConfigMap {
		return "configmap"
	}
	return "secret"
}

const (
	kindSecret    = "Secret"
	kindConfigMap = "ConfigMap"
)

// RecordDriver returns the driver a release record (a cached object labelled owner=helm) belongs
// to, by its kind: a ConfigMap's is DriverConfigMap, any other's DriverSecret.
func RecordDriver(o *unstructured.Unstructured) Driver {
	if o != nil && o.GetKind() == kindConfigMap {
		return DriverConfigMap
	}
	return DriverSecret
}

// DecodeConfigMap decodes the release a Helm release ConfigMap holds: data.release is Helm's own
// encoding as text (base64 of gzip of JSON, DecodePayload), without the Kubernetes base64 a
// Secret's data adds. The release's Labels come from the ConfigMap's labels, as Helm's driver sets
// them.
func DecodeConfigMap(cm *unstructured.Unstructured) (*rspb.Release, error) {
	data, _, _ := unstructured.NestedString(cm.Object, "data", "release")
	if data == "" {
		return nil, ErrNoPayload
	}
	rls, err := DecodePayload([]byte(data))
	if err != nil {
		return nil, fmt.Errorf("release ConfigMap %s/%s: %w", cm.GetNamespace(), cm.GetName(), err)
	}
	rls.Labels = cm.GetLabels()
	return rls, nil
}

// DecodeRecord decodes a release record of either driver, by its kind (RecordDriver).
func DecodeRecord(o *unstructured.Unstructured) (*rspb.Release, error) {
	if RecordDriver(o) == DriverConfigMap {
		return DecodeConfigMap(o)
	}
	return DecodeSecret(o)
}
