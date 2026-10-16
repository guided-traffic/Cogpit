package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// TestRealMemoryEstimateAgainstTheHeap checks the memory estimate on a real cluster's objects: it
// runs only when COGPIT_REAL_KUBECONFIG names the kubeconfig of a cluster the owner provides for
// tests (its current context; never in CI), reads a few kinds once (one list request each, nothing
// written), decodes each list's items into a store ten times as the informers do, and logs, per
// kind, the heap they take against the model's walk of every object and the size memo's first
// estimate. It logs numbers and kinds, never a name or an object.
func TestRealMemoryEstimateAgainstTheHeap(t *testing.T) {
	kubeconfig := os.Getenv("COGPIT_REAL_KUBECONFIG")
	if kubeconfig == "" {
		t.Skip("COGPIT_REAL_KUBECONFIG is not set: a real cluster's measurement")
	}
	if raceEnabled {
		t.Skip("the race detector's heap is laid out otherwise")
	}
	cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig}, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		t.Fatalf("the real cluster's kubeconfig: %v", err)
	}
	cfg.Timeout = time.Minute
	client, err := rest.HTTPClientFor(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	// Every list is read and turned into documents first, so that no response is still on the heap,
	// or freed, while a kind's copies are measured.
	paths := []string{"/api/v1/pods", "/api/v1/configmaps", "/api/v1/services", "/api/v1/events",
		"/apis/apps/v1/replicasets", "/apis/apps/v1/deployments", "/apis/apiextensions.k8s.io/v1/customresourcedefinitions"}
	docsOf := map[string][][]byte{}
	for _, path := range paths {
		raw, err := getJSON(ctx, client, strings.TrimRight(cfg.Host, "/")+path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		var list struct {
			APIVersion string `json:"apiVersion"`
			Kind       string `json:"kind"`
			Items      []json.RawMessage
		}
		if err := json.Unmarshal(raw, &list); err != nil {
			t.Fatalf("%s: the list does not parse", path)
		}
		for _, it := range list.Items {
			docsOf[path] = append(docsOf[path], asStored(t, it, list.APIVersion, strings.TrimSuffix(list.Kind, "List")))
		}
	}
	client.CloseIdleConnections()
	var lines []string
	var allMeasured, allWalked, allEstimated int64
	for _, path := range paths {
		docs := docsOf[path]
		if len(docs) == 0 {
			continue
		}
		measured, walked, estimated, n := decodeStore(t, docs, 10)
		kind := path[strings.LastIndex(path, "/")+1:]
		lines = append(lines, fmt.Sprintf("%-28s %5d objects, %7d B each on the heap, the model %.3f, the memo's first read %.3f",
			kind, n/10, measured/int64(n), float64(walked)/float64(measured), float64(estimated)/float64(measured)))
		allMeasured, allWalked, allEstimated = allMeasured+measured, allWalked+walked, allEstimated+estimated
	}
	t.Logf("the memory estimate on the real cluster's objects (each list decoded ten times):\n%s\nall: the model %.3f, the memo's first read %.3f",
		strings.Join(lines, "\n"), float64(allWalked)/float64(allMeasured), float64(allEstimated)/float64(allMeasured))
}

// getJSON reads a list as JSON, as the dynamic client the informers use does.
func getJSON(ctx context.Context, client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

// asStored is a list item's JSON as an informer stores the object: with its apiVersion and kind (a
// list's items carry neither), without metadata.managedFields (the cache's transform drops them).
// Errors never quote the object.
func asStored(t *testing.T, raw json.RawMessage, apiVersion, kind string) []byte {
	t.Helper()
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal("an item does not parse")
	}
	obj["apiVersion"], obj["kind"] = apiVersion, kind
	if md, ok := obj["metadata"].(map[string]any); ok {
		delete(md, "managedFields")
	}
	out, err := json.Marshal(obj)
	if err != nil {
		t.Fatal("an item does not encode")
	}
	return out
}
