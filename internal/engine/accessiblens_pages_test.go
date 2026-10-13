package engine

import (
	"context"
	"testing"

	"github.com/guided-traffic/cogpit/internal/testkit"
)

// podsHandles counts the handles on pods in all namespaces and on pods of single namespaces.
func podsHandles(s *Session) (clusterWide, namespaced int) {
	for _, st := range s.Backend().Cache.Stats() {
		if st.Scope.GVR != testkit.PodsGVR {
			continue
		}
		if st.Scope.Namespace == "" {
			clusterWide += st.Handles
		} else {
			namespaced += st.Handles
		}
	}
	return clusterWide, namespaced
}

// A page opened while the access reviews are still out takes its kinds' handles as a pending plan
// reads them, in all namespaces. Once the reviews say the user may not list pods there, the
// Workloads Overview and the cluster Overview read pods again namespace by namespace, as the Pods
// view and the drawer do: no cluster-wide pods handle is left (the API server refuses it), and the
// Workloads Overview counts the accessible namespaces' pods without an error.
func TestPagesFollowTheAccessReviews(t *testing.T) {
	for _, page := range []string{WorkloadsOverview, ClusterOverview} {
		t.Run(page, func(t *testing.T) {
			_, src, rev := namespacedOnly(t)
			rev.set(func(f *fakeReviewer) { f.gate = make(chan struct{}) })
			s := nsSession(t, src, rev, "team-a", "team-a", "team-b")
			connectTest(t, s)
			s.UpdateUI(func(u *UIState) { u.View, u.Sort = page, s.ViewSort(page) })
			waitFor(t, "the page's handle on pods", func() bool { all, _ := podsHandles(s); return all > 0 })

			rev.set(func(f *fakeReviewer) { close(f.gate); f.gate = nil })
			waitFor(t, "the page reading pods namespace by namespace", func() bool {
				all, ns := podsHandles(s)
				return all == 0 && ns > 0
			})
			if page != WorkloadsOverview {
				return
			}
			waitFor(t, "the accessible namespaces' pods counted", func() bool {
				m, _, err := s.WorkloadsOverview(context.Background(), nil, false)
				if err != nil {
					t.Fatal(err)
				}
				return m.Synced && len(m.Kinds) > 0 && m.Kinds[0].Kind == "pods" && m.Kinds[0].Count == 2 && m.Kinds[0].Error == ""
			})
		})
	}
}
