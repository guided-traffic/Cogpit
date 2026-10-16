package cluster

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/restmapper"
)

// Mapping is what discovery says about a kind: the resource that serves it in the requested (or
// preferred) version, its Kubernetes kind, and whether its objects live in namespaces.
type Mapping struct {
	GVR        schema.GroupVersionResource `json:"gvr"`
	GVK        schema.GroupVersionKind     `json:"gvk"`
	Namespaced bool                        `json:"namespaced"`
}

// ErrNoPrecondition refuses an update that carries no resourceVersion: without it the API server
// would overwrite whatever the object holds now, which Cogpit never does.
var ErrNoPrecondition = errors.New("an update needs the resourceVersion it was based on")

// objectMapper is a Conn's REST mapper for the generic object writes: built on first use over the
// connection's cached discovery (discocache.go: the answer ServedResources uses, from memory or the
// disk cache), so that mapping a kind costs no request when connecting asked already, and reset
// when a kind is not found, because a custom resource definition may have been added since, and
// when the definitions changed (DefinitionsSeen).
type objectMapper struct {
	mu     sync.Mutex
	mapper *restmapper.DeferredDiscoveryRESTMapper
}

func (c *Conn) restMapper() (*restmapper.DeferredDiscoveryRESTMapper, error) {
	c.objects.mu.Lock()
	defer c.objects.mu.Unlock()
	if c.objects.mapper != nil {
		return c.objects.mapper, nil
	}
	cached, _, err := c.discovery()
	if err != nil {
		return nil, err
	}
	c.objects.mapper = restmapper.NewDeferredDiscoveryRESTMapperWithContext(cached)
	return c.objects.mapper, nil
}

// resetMapper drops the REST mapper's mappings; the next mapping builds them again from discovery.
func (c *Conn) resetMapper() {
	c.objects.mu.Lock()
	c.objects.mapper = nil
	c.objects.mu.Unlock()
}

// MapKind maps an apiVersion and kind, as a manifest names them, to the resource that serves
// them. A kind the cached discovery does not know triggers one fresh discovery round before the
// error, so a custom resource definition created a moment ago is found.
func (c *Conn) MapKind(ctx context.Context, apiVersion, kind string) (Mapping, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return Mapping{}, fmt.Errorf("apiVersion %q: %w", apiVersion, err)
	}
	if kind == "" || gv.Version == "" {
		return Mapping{}, fmt.Errorf("apiVersion %q and kind %q do not name a kind", apiVersion, kind)
	}
	m, err := c.restMapper()
	if err != nil {
		return Mapping{}, err
	}
	gk := schema.GroupKind{Group: gv.Group, Kind: kind}
	rm, err := m.RESTMappingWithContext(ctx, gk, gv.Version)
	if meta.IsNoMatchError(err) {
		m.ResetWithContext(ctx)
		rm, err = m.RESTMappingWithContext(ctx, gk, gv.Version)
	}
	if err != nil {
		return Mapping{}, fmt.Errorf("context %q: %s %s: %w", c.contextName, apiVersion, kind, err)
	}
	return mappingOf(rm), nil
}

// MapResource maps a resource, with or without its group ("configmaps", "gateways" in
// "gateway.networking.k8s.io"), to its preferred version, its kind and its scope. An empty group
// matches the resource in any group, the core group first.
func (c *Conn) MapResource(ctx context.Context, group, resource string) (Mapping, error) {
	if resource == "" {
		return Mapping{}, errors.New("no resource given")
	}
	m, err := c.restMapper()
	if err != nil {
		return Mapping{}, err
	}
	partial := schema.GroupVersionResource{Group: group, Resource: strings.ToLower(resource)}
	gvr, err := m.ResourceForWithContext(ctx, partial)
	if meta.IsNoMatchError(err) {
		m.ResetWithContext(ctx)
		gvr, err = m.ResourceForWithContext(ctx, partial)
	}
	if err != nil {
		return Mapping{}, fmt.Errorf("context %q: resource %s: %w", c.contextName, partial.GroupResource(), err)
	}
	gvk, err := m.KindForWithContext(ctx, gvr)
	if err != nil {
		return Mapping{}, fmt.Errorf("context %q: resource %s: %w", c.contextName, gvr.GroupResource(), err)
	}
	rm, err := m.RESTMappingWithContext(ctx, gvk.GroupKind(), gvk.Version)
	if err != nil {
		return Mapping{}, fmt.Errorf("context %q: resource %s: %w", c.contextName, gvr.GroupResource(), err)
	}
	return mappingOf(rm), nil
}

func mappingOf(rm *meta.RESTMapping) Mapping {
	return Mapping{
		GVR:        rm.Resource,
		GVK:        rm.GroupVersionKind,
		Namespaced: rm.Scope != nil && rm.Scope.Name() == meta.RESTScopeNameNamespace,
	}
}

// UpdateObject replaces an object with obj (HTTP PUT) through the dynamic client, as Cogpit's
// field manager. obj must carry the resourceVersion the edit started from: the API server refuses
// the update with 409 Conflict when the object has changed since, so a concurrent change is never
// overwritten. With dryRun the server validates and admits the update (dryRun=All) without
// storing it. It returns the object as the server stored it, or would store it.
func (c *Conn) UpdateObject(ctx context.Context, gvr schema.GroupVersionResource, obj *unstructured.Unstructured, dryRun bool) (*unstructured.Unstructured, error) {
	if obj == nil {
		return nil, errors.New("update: no object")
	}
	what := describeObject(gvr, obj.GetNamespace(), obj.GetName())
	if obj.GetResourceVersion() == "" {
		return nil, fmt.Errorf("update %s: %w", what, ErrNoPrecondition)
	}
	opts := metav1.UpdateOptions{FieldManager: FieldManager}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}
	out, err := c.dyn.Resource(gvr).Namespace(obj.GetNamespace()).Update(ctx, obj, opts)
	if err != nil {
		return nil, fmt.Errorf("update %s: %w", what, err)
	}
	return out, nil
}

// CreateObject creates obj in namespace (empty for a cluster-scoped kind) through the dynamic
// client, as Cogpit's field manager. With dryRun the server validates and admits it
// (dryRun=All) without storing it. It returns the object as the server stored it, or would.
func (c *Conn) CreateObject(ctx context.Context, gvr schema.GroupVersionResource, namespace string, obj *unstructured.Unstructured, dryRun bool) (*unstructured.Unstructured, error) {
	if obj == nil {
		return nil, errors.New("create: no object")
	}
	name := obj.GetName()
	if name == "" {
		name = obj.GetGenerateName() + "*"
	}
	what := describeObject(gvr, namespace, name)
	opts := metav1.CreateOptions{FieldManager: FieldManager}
	if dryRun {
		opts.DryRun = []string{metav1.DryRunAll}
	}
	out, err := c.dyn.Resource(gvr).Namespace(namespace).Create(ctx, obj, opts)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", what, err)
	}
	return out, nil
}

// DeleteObject deletes one object of any kind with its default grace period and the default
// propagation (dependents are garbage-collected in the background). A non-empty uid is a
// precondition: the API server refuses the delete if the object with that name is another object
// by now (one created again under the same name while the user confirmed).
func (c *Conn) DeleteObject(ctx context.Context, gvr schema.GroupVersionResource, namespace, name, uid string) error {
	opts := metav1.DeleteOptions{}
	if uid != "" {
		u := types.UID(uid)
		opts.Preconditions = &metav1.Preconditions{UID: &u}
	}
	if err := c.dyn.Resource(gvr).Namespace(namespace).Delete(ctx, name, opts); err != nil {
		return fmt.Errorf("delete %s: %w", describeObject(gvr, namespace, name), err)
	}
	return nil
}

// describeObject names an object in errors the way kubectl names resources: "configmaps
// team-a/web", "deployments.apps team-a/web". It never includes the object's content.
func describeObject(gvr schema.GroupVersionResource, namespace, name string) string {
	res := gvr.Resource
	if gvr.Group != "" {
		res += "." + gvr.Group
	}
	if namespace != "" {
		return res + " " + namespace + "/" + name
	}
	return res + " " + name
}
