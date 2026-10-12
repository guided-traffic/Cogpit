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

// SourcesForConn returns a connection's two providers for NewHubFor: the Metrics Server for the
// current values, and Prometheus through the service proxy, found among services (or named by the
// override set on the returned PrometheusDetector), for history. All requests go through the
// connection's clients, so they are counted and rate limited.
func SourcesForConn(conn *cluster.Conn, services ServiceSource, opts PrometheusOptions) (Sources, *PrometheusDetector, error) {
	server, err := ForConn(conn)
	if err != nil {
		return Sources{}, nil, err
	}
	prom := NewPrometheusDetector(conn.Kube(), services, opts)
	return Sources{Live: server, History: prom}, prom, nil
}

// ChainForConn returns one detector for a connection: Prometheus through the service proxy if one
// is found among services (or named by the override set on the returned PrometheusDetector), else
// the Metrics Server; a hub over it (NewHub) serves everything from the one it found. The engine
// uses SourcesForConn, both together; the chain remains for a single provider. All requests go
// through the connection's clients, so they are counted and rate limited.
func ChainForConn(conn *cluster.Conn, services ServiceSource, opts PrometheusOptions) (Detector, *PrometheusDetector, error) {
	server, err := ForConn(conn)
	if err != nil {
		return nil, nil, err
	}
	prom := NewPrometheusDetector(conn.Kube(), services, opts)
	return Chain(prom, server), prom, nil
}
