//go:build !nodemo

package main

import (
	"github.com/guided-traffic/cogpit/internal/demo"
	"github.com/guided-traffic/cogpit/internal/engine"
)

// testDemo sizes the tests' demo cluster, the context "demo" of a development build: small and
// still, so that every run reads the same objects.
func testDemo() engine.DemoOptions {
	return demo.Options{Pods: 40, Services: 6, Namespaces: 3, Nodes: 3, Deployments: 4, Events: 20,
		NoChurn: true, Seed: 7}
}
