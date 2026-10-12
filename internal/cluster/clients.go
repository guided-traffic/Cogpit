package cluster

import (
	"net/http"

	"k8s.io/client-go/discovery"
)

// HTTPClient returns the connection's HTTP client: the one pool (one HTTP/2 connection) every
// client of the connection uses, behind the counting transport. Packages that build a typed client
// of their own for an aggregated API (the Metrics API) pass it to the client's
// NewForConfigAndClient together with RESTConfig, so that their requests are counted and share
// the connection's rate limit.
func (c *Conn) HTTPClient() *http.Client { return c.httpClient }

// Discovery returns the connection's discovery client, with the discovery budget of its own and
// behind the counting transport.
func (c *Conn) Discovery() discovery.DiscoveryInterface { return c.disco }
