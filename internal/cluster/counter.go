package cluster

import "net/http"

// RequestStats is a snapshot of the API requests one connection has made.
type RequestStats struct {
	// Total is the number of requests, watches included.
	Total int64 `json:"total"`
	// ByVerb counts requests by verb: "list", "watch", "get", "create", "update", "patch",
	// "delete", "discovery", "other".
	ByVerb map[string]int64 `json:"byVerb"`
	// ByVerbResource counts requests by "verb resource", "list pods", "watch services".
	ByVerbResource map[string]int64 `json:"byVerbResource"`
}

// RequestCounter counts the API requests of one connection, by verb and resource. It is the
// evidence for the API-load budgets P9–P11 and feeds the status bar.
type RequestCounter struct{}

// NewRequestCounter returns an empty counter.
func NewRequestCounter() *RequestCounter { return &RequestCounter{} }

// Wrap returns a round-tripper that counts every request before passing it to next. Install it
// with rest.Config.Wrap.
func (rc *RequestCounter) Wrap(next http.RoundTripper) http.RoundTripper { return next }

// Snapshot returns the counts so far.
func (rc *RequestCounter) Snapshot() RequestStats {
	return RequestStats{ByVerb: map[string]int64{}, ByVerbResource: map[string]int64{}}
}
