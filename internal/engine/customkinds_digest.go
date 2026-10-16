package engine

import (
	"crypto/sha256"
	"encoding/hex"
	"strconv"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// The custom resource definitions confirm the connection's discovery answer (D-65, the discovery
// cache on disk): each rebuild of the custom kinds from a complete list of the definitions hands
// their digest to the connection (cluster.Conn.DefinitionsSeen), which drops an answer checked
// against other definitions, or read from a file that nobody checked, so that the next read asks
// the server. A user who may not list the definitions has no digest: such an answer is reused for
// cluster.ServedResourcesTTL only, and a refresh of the access reviews asks again (retake).

// definitionsDigest is the digest of the listed definitions, sorted by name: each one's name,
// generation (a change of its versions, names or scope moves it), whether it is established and
// whether it is being deleted, the facts that decide what discovery serves for it.
func definitionsDigest(crds []*unstructured.Unstructured) string {
	h := sha256.New()
	for _, crd := range crds {
		line := crd.GetName() + " " + strconv.FormatInt(crd.GetGeneration(), 10) + " " +
			strconv.FormatBool(established(crd)) + " " + strconv.FormatBool(crd.GetDeletionTimestamp() != nil) + "\n"
		_, _ = h.Write([]byte(line))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// established reports whether a definition's Established condition is True.
func established(crd *unstructured.Unstructured) bool {
	conds, _, _ := unstructured.NestedSlice(crd.Object, "status", "conditions")
	for _, c := range conds {
		m, ok := c.(map[string]any)
		if ok && m["type"] == "Established" {
			return m["status"] == condTrue
		}
	}
	return false
}

// confirmDiscovery hands the definitions' digest to the connection after a rebuild from their
// complete list (synced, no error).
func (ck *customKinds) confirmDiscovery(crds []*unstructured.Unstructured, synced bool, errMsg string) {
	if !synced || errMsg != "" || ck.b.Conn == nil {
		return
	}
	ck.b.Conn.DefinitionsSeen(definitionsDigest(crds))
}
