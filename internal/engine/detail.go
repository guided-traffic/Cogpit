package engine

import (
	"github.com/guided-traffic/cogpit/internal/cache"
	"github.com/guided-traffic/cogpit/internal/cluster"
	"github.com/guided-traffic/cogpit/internal/model"
)

// DetailModel is what the detail drawer shows for one object, built from the cache.
type DetailModel struct {
	Ref    model.ObjectRef `json:"ref"`
	Title  string          `json:"title"` // "Pod: web-0"
	Status model.Status    `json:"status,omitempty"`
	// Sections: the generic metadata section first, then the kind's own sections.
	Sections []model.DetailSection `json:"sections"`
	// Events are the events whose involved object is this object, newest first.
	Events []EventItem `json:"events"`
	// YAML is the object as YAML, managedFields dropped, Secret values redacted.
	YAML string `json:"yaml"`
	// Containers are the pod's container names (logs and shell pick one); empty for other kinds.
	Containers []string `json:"containers,omitempty"`
	// Missing is true when the object is not (or no longer) in the cache.
	Missing bool `json:"missing,omitempty"`
}

// EventItem is one event in the drawer.
type EventItem struct {
	Type     string       `json:"type"` // Normal, Warning
	Reason   string       `json:"reason"`
	Message  string       `json:"message"`
	Source   string       `json:"source,omitempty"`
	Count    int64        `json:"count,omitempty"`
	LastSeen int64        `json:"lastSeen,omitempty"` // Unix seconds
	Status   model.Status `json:"status,omitempty"`
}

// Diagnostics are the counters the status bar and the diagnostics view show.
type Diagnostics struct {
	Context   string                `json:"context"`
	Demo      bool                  `json:"demo,omitempty"`
	Requests  cluster.RequestStats  `json:"requests"`
	Informers []cache.InformerStats `json:"informers"`
	// Snapshots counts table snapshots built since start, for the coalescing evidence.
	Snapshots uint64 `json:"snapshots"`
	DemoStats any    `json:"demoStats,omitempty"`
}
