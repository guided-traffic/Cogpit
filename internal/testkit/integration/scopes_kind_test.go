//go:build kind

package integration_test

import (
	"context"
	"os"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/testkit"
)

// TestKindSecretScopes checks the Secrets contract of D-13 against a real API server: a
// metadata-only informer (PartialObjectMetadata, streamed or listed) holds names and labels but no
// value, and a label-selected informer holds only the Secrets it selects, in full. It creates two
// Secrets in its own namespace, cogpit-scopes, and deletes the namespace afterwards.
func TestKindSecretScopes(t *testing.T) {
	kubeconfig, contextName := os.Getenv("COGPIT_KIND_KUBECONFIG"), os.Getenv("COGPIT_KIND_CONTEXT")
	if kubeconfig == "" || contextName == "" {
		t.Skip("COGPIT_KIND_KUBECONFIG and COGPIT_KIND_CONTEXT are not set; run make test-kind")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	conn, err := cluster.Connect(ctx, cluster.Options{KubeconfigPaths: []string{kubeconfig}, Context: contextName})
	if err != nil {
		t.Fatalf("Connect: %v", err)
	}
	defer conn.Close()

	const ns = "cogpit-scopes"
	kube := conn.Kube()
	_ = kube.CoreV1().Namespaces().Delete(ctx, ns, metav1.DeleteOptions{})
	if !testkit.WaitFor(ctx, func() bool {
		_, err := kube.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}, metav1.CreateOptions{})
		return err == nil
	}) {
		t.Fatal("cannot create the namespace")
	}
	t.Cleanup(func() {
		_ = kube.CoreV1().Namespaces().Delete(context.Background(), ns, metav1.DeleteOptions{})
	})
	for _, s := range []*corev1.Secret{
		{ObjectMeta: metav1.ObjectMeta{Name: "plain", Namespace: ns}, StringData: map[string]string{"password": "hunter2"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.x.v1", Namespace: ns, Labels: map[string]string{"owner": "helm", "name": "x"}},
			Type: "helm.sh/release.v1", StringData: map[string]string{"release": "payload"}},
	} {
		if _, err := kube.CoreV1().Secrets(ns).Create(ctx, s, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	c := cache.New(cache.DynamicSource(conn.Dynamic(), conn.Metadata()), cache.Options{})
	defer c.Close()
	meta := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, Namespace: ns, MetadataOnly: true})
	defer meta.Release()
	helm := c.Acquire(cache.Scope{GVR: testkit.SecretsGVR, Namespace: ns, LabelSelector: "owner=helm"})
	defer helm.Release()
	for _, h := range []*cache.Handle{meta, helm} {
		if !h.WaitForSync(ctx) {
			t.Fatalf("%v not synced: %v", h.Scope(), h.Err())
		}
	}
	if !testkit.WaitFor(ctx, func() bool { return len(meta.List()) >= 2 }) {
		t.Fatalf("metadata scope: %d Secrets", len(meta.List()))
	}
	for _, o := range meta.List() {
		if _, has := o.Object["data"]; has || !cache.IsMetadataObject(o) || o.GetUID() == "" {
			t.Errorf("metadata scope holds %v", o.Object)
		}
	}
	if l := helm.List(); len(l) != 1 || l[0].GetName() != "sh.helm.release.v1.x.v1" {
		t.Fatalf("owner=helm scope: %d objects", len(l))
	} else if _, has := l[0].Object["data"]; !has {
		t.Error("owner=helm scope lost the payload")
	}
	st := conn.Counter().Snapshot()
	t.Logf("requests: %v", st.ByVerbResource)
}
