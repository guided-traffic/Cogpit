package kinds_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func FuzzDaemonSets(f *testing.F) { fuzzProfile(f, "daemonsets") }

func TestDaemonSetsStatusTexts(t *testing.T) {
	const none = model.StatusNone
	checkStatusTexts(t, []statusText{
		{"daemonsets", "healthy", "pods", "3/3", none, model.StatusOK},
		{"daemonsets", "healthy", "nodeSelector", "kubernetes.io/os=linux", none, model.StatusOK},
		{"daemonsets", "failing", "pods", "1/3", none, model.StatusWarning},
		{"daemonsets", "failing", "nodeSelector", "kubernetes.io/os=linux, node.example.com/pool=general", none, model.StatusWarning},
		{"daemonsets", "rolling-update", "pods", "3/4", none, model.StatusPending},
		{"daemonsets", "on-delete", "pods", "2/3", none, model.StatusWarning}, // OnDelete replaces nothing by itself
		{"daemonsets", "no-matching-nodes", "pods", "0/0", none, none},
		{"daemonsets", "minimal", "nodeSelector", "", none, none},
		{"daemonsets", "terminating", "pods", "2/2", none, model.StatusTerminating},
	})
}
