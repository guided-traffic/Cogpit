package cluster

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// secretsGVR is the core Secrets resource.
var secretsGVR = schema.GroupVersionResource{Version: "v1", Resource: "secrets"}

// GetSecret reads one Secret, data included, with a direct GET (get secrets, counted by the
// connection's request counter). It is the one read of Secret values Cogpit makes: the cache holds
// Secrets as metadata only (D-13, H-8), and the drawer fetches the Secret it shows. Errors name the
// Secret, never its content.
func (c *Conn) GetSecret(ctx context.Context, namespace, name string) (*unstructured.Unstructured, error) {
	if namespace == "" || name == "" {
		return nil, errors.New("get secret: a namespace and a name are needed")
	}
	if c.dyn == nil {
		return nil, fmt.Errorf("context %q: no client to read Secrets with", c.contextName)
	}
	obj, err := c.dyn.Resource(secretsGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("get %s: %w", describeObject(secretsGVR, namespace, name), err)
	}
	return obj, nil
}
