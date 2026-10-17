package engine

import "testing"

// The Overview's Issues date a node's problem as the Nodes list reads its conditions: a condition
// whose type ends in Ready (SystemdUnitsReady, the owner's finding of 2026-10-17) is a problem when
// False or Unknown, any other when True.
func TestNodeConditionProblem(t *testing.T) {
	for _, c := range []struct {
		typ, status string
		want        bool
	}{
		{"Ready", "True", false},
		{"Ready", "False", true},
		{"Ready", "Unknown", true},
		{"SystemdUnitsReady", "True", false},
		{"SystemdUnitsReady", "False", true},
		{"SystemdUnitsReady", "Unknown", true},
		{"MemoryPressure", "True", true},
		{"MemoryPressure", "False", false},
		{"KernelDeadlock", "True", true},
		{"NetworkUnavailable", "False", false},
	} {
		if got := nodeConditionProblem(c.typ, c.status); got != c.want {
			t.Errorf("%s %s: %v, want %v", c.typ, c.status, got, c.want)
		}
	}
}
