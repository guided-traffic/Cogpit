package engine

import (
	"fmt"
	"slices"
	"strings"

	"k8s.io/apimachinery/pkg/util/validation"
)

// Accessible namespaces (G3, G13; docs/parity/ui-parity.md › Cluster settings): Lens's cluster
// setting for a user who may not list the cluster's namespaces, kept per context in
// cluster-settings.json (ClusterSettings.AccessibleNamespaces). It only narrows what Cogpit reads:
// the namespace filter offers these namespaces when Namespaces cannot be listed, and a namespaced
// kind the user may not list in all namespaces is watched in each of them instead
// (accessiblens_plan.go). It writes nothing to a cluster or a kubeconfig, so agents set it freely
// (cluster.settings.set, a UI command).

// maxAccessibleNamespaces bounds the list: each namespace may cost one watch per open kind
// (Cogpit's bound, not the owner's decision).
const maxAccessibleNamespaces = 100

// checkAccessibleNamespaces refuses a list Cogpit would not keep: a name that is no namespace's
// (DNS-1123 label), or too many.
func checkAccessibleNamespaces(names []string) error {
	if len(names) > maxAccessibleNamespaces {
		return fmt.Errorf("at most %d accessible namespaces", maxAccessibleNamespaces)
	}
	for _, ns := range names {
		if errs := validation.IsDNS1123Label(ns); len(errs) > 0 {
			return fmt.Errorf("accessible namespace %q: %s", ns, strings.Join(errs, "; "))
		}
	}
	return nil
}

// sortedAccessibleNamespaces trims the names, drops empty ones and duplicates, and sorts them:
// the form the settings keep (Cogpit's: the order of adding is not kept).
func sortedAccessibleNamespaces(names []string) []string {
	if len(names) == 0 {
		return nil
	}
	out := make([]string, 0, len(names))
	for _, ns := range names {
		if ns = strings.TrimSpace(ns); ns != "" {
			out = append(out, ns)
		}
	}
	slices.Sort(out)
	out = slices.Compact(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

// keptAccessibleNamespaces is what a settings file's list becomes on load: the valid names,
// sorted, at most maxAccessibleNamespaces.
func keptAccessibleNamespaces(names []string) []string {
	var out []string
	for _, ns := range sortedAccessibleNamespaces(names) {
		if len(validation.IsDNS1123Label(ns)) == 0 && len(out) < maxAccessibleNamespaces {
			out = append(out, ns)
		}
	}
	return out
}
