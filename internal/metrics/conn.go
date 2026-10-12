package metrics

import (
	"fmt"

	metricsclient "k8s.io/metrics/pkg/client/clientset/versioned/typed/metrics/v1beta1"

	"github.com/guided-traffic/cogpit/internal/cluster"
)

// ForConn returns the Metrics Server provider of a connection. Its requests go through the
// connection's HTTP client, so the counting transport sees them (budget P12) and the cluster's
// rate limit bounds them.
func ForConn(conn *cluster.Conn) (*Server, error) {
	client, err := metricsclient.NewForConfigAndClient(conn.RESTConfig(), conn.HTTPClient())
	if err != nil {
		return nil, fmt.Errorf("metrics client: %w", err)
	}
	return NewServer(conn.Discovery(), client), nil
}
