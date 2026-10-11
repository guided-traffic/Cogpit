package helm

import (
	"errors"
	"fmt"
	"log/slog"

	"helm.sh/helm/v4/pkg/action"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// NewActionConfig returns the Helm SDK's action configuration for one namespace of a connection,
// on the connection's own client configuration (its credentials, rate limiter and request
// counter: pass cluster.Conn.RESTConfig()) and Helm's default Secrets storage driver. It never
// reads or writes a kubeconfig file: the SDK sees only cfg.
func NewActionConfig(cfg *rest.Config, namespace string, log *slog.Logger) (*action.Configuration, error) {
	if cfg == nil {
		return nil, errors.New("no client configuration")
	}
	if log == nil {
		log = slog.Default()
	}
	ac := action.NewConfiguration()
	ac.SetLogger(log.Handler())
	if err := ac.Init(newRESTGetter(cfg, namespace), namespace, "secret"); err != nil {
		return nil, fmt.Errorf("helm: %w", err)
	}
	return ac, nil
}

// restGetter is the RESTClientGetter the Helm SDK builds its clients from: a fixed client
// configuration and namespace instead of kubeconfig flags.
type restGetter struct {
	cfg       *rest.Config
	namespace string
	disco     discovery.CachedDiscoveryInterface
}

func newRESTGetter(cfg *rest.Config, namespace string) *restGetter {
	return &restGetter{cfg: rest.CopyConfig(cfg), namespace: namespace}
}

func (g *restGetter) ToRESTConfig() (*rest.Config, error) { return rest.CopyConfig(g.cfg), nil }

func (g *restGetter) ToDiscoveryClient() (discovery.CachedDiscoveryInterface, error) {
	if g.disco == nil {
		dc, err := discovery.NewDiscoveryClientForConfig(g.cfg)
		if err != nil {
			return nil, err
		}
		g.disco = memory.NewMemCacheClient(dc)
	}
	return g.disco, nil
}

func (g *restGetter) ToRESTMapper() (meta.RESTMapper, error) {
	dc, err := g.ToDiscoveryClient()
	if err != nil {
		return nil, err
	}
	return restmapper.NewDeferredDiscoveryRESTMapper(dc), nil
}

func (g *restGetter) ToRawKubeConfigLoader() clientcmd.ClientConfig {
	return fixedClientConfig{getter: g}
}

// fixedClientConfig answers the SDK's kubeconfig questions from the fixed configuration; it has no
// file behind it, so ConfigAccess, through which a kubeconfig could be written, is nil.
type fixedClientConfig struct{ getter *restGetter }

func (c fixedClientConfig) RawConfig() (clientcmdapi.Config, error) {
	return clientcmdapi.Config{}, nil
}

func (c fixedClientConfig) ClientConfig() (*rest.Config, error) { return c.getter.ToRESTConfig() }

func (c fixedClientConfig) Namespace() (string, bool, error) {
	return c.getter.namespace, false, nil
}

func (c fixedClientConfig) ConfigAccess() clientcmd.ConfigAccess { return nil }
