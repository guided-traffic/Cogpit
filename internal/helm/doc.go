// Package helm is Cogpit's Helm integration on the Helm SDK (helm.sh/helm/v4), in-process
// (D-3): Helm releases read from the release Secrets the cache holds (a watch on the Secrets
// labelled owner=helm, D-13), and the release actions that write, crash-safe (D-11).
//
// Layer L2: it imports model (L0) and the cache and cluster layers (L1), and only it talks to the
// Helm SDK.
package helm
