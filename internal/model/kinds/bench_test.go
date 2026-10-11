package kinds_test

import (
	"strconv"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// BenchmarkBuildRowPods builds the rows of 10,000 pods, a mix of the recorded states with
// distinct names. The engine rebuilds a row on every watch event, so the per-pod cost bounds the
// event rate a view can follow. One op is the whole list; ns/pod is reported alongside.
func BenchmarkBuildRowPods(b *testing.B) {
	const n = 10000
	p, _ := kinds.ByKey("pods")
	cases := loadCases(b, "pods")
	objs := make([]*unstructured.Unstructured, n)
	for i := range objs {
		obj := cases[i%len(cases)].obj.DeepCopy()
		obj.SetName("pod-" + strconv.Itoa(i))
		objs[i] = obj
	}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, obj := range objs {
			_ = model.BuildRow(p, "bench", obj)
		}
	}
	b.ReportMetric(float64(b.Elapsed().Nanoseconds())/float64(b.N)/n, "ns/pod")
}
