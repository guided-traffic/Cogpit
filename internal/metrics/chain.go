package metrics

import (
	"context"
	"encoding/json"
)

// isBuildInfo reports whether a body is Prometheus's buildinfo answer.
func isBuildInfo(body []byte) bool {
	var r struct {
		Status string `json:"status"`
		Data   struct {
			Version string `json:"version"`
		} `json:"data"`
	}
	return json.Unmarshal(body, &r) == nil && r.Status == "success" && r.Data.Version != ""
}

// Chain is the detector that tries each of dets in order and takes the first provider that is
// found: Prometheus, then the Metrics Server. When none is found it reports the most telling
// state: a forbidden or unavailable one before "not served", so a cluster with a Prometheus the
// user may not read says so instead of "no metrics-server".
func Chain(dets ...Detector) Detector {
	return DetectorFunc(func(ctx context.Context) (Provider, State) {
		var best State
		for _, d := range dets {
			p, st := d.Detect(ctx)
			if p != nil && st.Usable() {
				return p, st
			}
			if ctx.Err() != nil {
				return nil, st
			}
			if best.Status == "" || severity(st.Status) > severity(best.Status) {
				best = st
			}
		}
		if best.Status == "" {
			best = State{Status: StatusNotServed, Message: "no metrics provider"}
		}
		return nil, best
	})
}

func severity(s Status) int {
	switch s {
	case StatusForbidden:
		return 3
	case StatusUnavailable:
		return 2
	case StatusNotServed:
		return 1
	}
	return 0
}
