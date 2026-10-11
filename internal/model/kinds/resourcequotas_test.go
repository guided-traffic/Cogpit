package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

func FuzzResourceQuotas(f *testing.F) { fuzzProfile(f, "resourcequotas") }

func TestResourceQuotasStatusTexts(t *testing.T) {
	const none = model.StatusNone
	// The columns are Lens's documented ones; the status rule is inferred.
	checkStatusTexts(t, []statusText{
		{"resourcequotas", "healthy", "name", "compute", none, model.StatusOK},
		{"resourcequotas", "healthy", "namespace", "payments", none, model.StatusOK},
		{"resourcequotas", "failing", "age", "10d", none, model.StatusWarning},
		{"resourcequotas", "scoped", "name", "high-priority", none, model.StatusPending},
		{"resourcequotas", "minimal", "name", "empty", none, none},
		{"resourcequotas", "terminating", "name", "object-counts", none, model.StatusTerminating},
	})
}

// TestResourceQuotaAtLimit pins which resources count as run out: pods at the limit and memory
// over a lowered limit do; CPU below the limit and a limit of zero that nothing uses do not.
func TestResourceQuotaAtLimit(t *testing.T) {
	p, _ := kinds.ByKey("resourcequotas")
	for _, c := range loadCases(t, "resourcequotas") {
		got := model.FieldValues(p, "atLimit", c.obj)
		want := map[string][]string{"failing": {"pods", "requests.memory"}}[c.name]
		if len(got) != len(want) {
			t.Errorf("%s: at limit %q, want %q", c.name, got, want)
			continue
		}
		for i := range got {
			if got[i] != want[i] {
				t.Errorf("%s: at limit %q, want %q", c.name, got, want)
			}
		}
	}
}
