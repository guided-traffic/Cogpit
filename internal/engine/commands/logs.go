package commands

import (
	"context"
	"fmt"
	"slices"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// logOptionsParams are logs.setOptions' parameters: the tab and the options to change; options
// left out keep their value.
type logOptionsParams struct {
	ID            string  `json:"id" jsonschema:"logs tab id as ui_state lists it, e.g. logs-1"`
	Previous      *bool   `json:"previous,omitempty" jsonschema:"show the log of the container's previous, terminated instance (the stream opens again)"`
	Timestamps    *bool   `json:"timestamps,omitempty" jsonschema:"start each line with the API server's RFC 3339 timestamp (the stream opens again)"`
	ResourceNames *bool   `json:"resourceNames,omitempty" jsonschema:"show each line's pod and container, in tabs of a controlling resource"`
	Wrap          *bool   `json:"wrap,omitempty" jsonschema:"wrap long lines instead of cutting them at the tab's edge"`
	Search        *string `json:"search,omitempty" jsonschema:"text to find in the lines, case-insensitively, at most 1024 bytes; empty clears it"`
	Filter        *bool   `json:"filter,omitempty" jsonschema:"show only the lines that contain search; off shows every line with the matches highlighted"`
}

// workloadLogVerbs are what a controlling resource's logs cost: the pods' (and for a Deployment
// the replica sets') list and watch through the cache, and each container's log.
var workloadLogVerbs = []engine.Verb{
	{Verb: nwVerbGet, Resource: kindPods, Subresource: "log"},
	{Verb: nwVerbList, Resource: kindPods}, {Verb: nwVerbWatch, Resource: kindPods},
	{Verb: nwVerbList, Group: nwGroupApps, Resource: replicaSetKindKey}, {Verb: nwVerbWatch, Group: nwGroupApps, Resource: replicaSetKindKey},
}

func registerLogs(g *registrar) {
	// Lens's documented orders: Deployment Scale, Restart, Logs (captured: 300); Stateful Set
	// Restart, Scale, Logs; Daemon Set Restart, Logs; Replica Set Scale, Logs; Job Logs. Logs
	// after the kind's other actions everywhere.
	add(g, engine.Spec[refParams]{
		ID: "workload.logs", Label: "Logs", MenuPath: "Workload/Logs", Kinds: engine.WorkloadLogKinds, Order: 300,
		Danger: engine.DangerRead, Verbs: workloadLogVerbs,
		Description: "Opens a dock tab with the logs of every container of every pod a Deployment, Daemon Set, Stateful " +
			"Set, Replica Set or Job controls (a Deployment's through its replica sets), each line starting with " +
			"<pod>/<container>, and follows them: pods and restarted containers join as they start. It starts with up " +
			"to 1,000 lines of history shared by the containers and streams at most 40 containers at once. Activates " +
			"the tab already open on the resource. Options: logs_setOptions. Returns the tab.",
		Enabled: func(s *engine.Session, p refParams) error {
			if !slices.Contains(engine.WorkloadLogKinds, p.Ref.Kind) {
				return fmt.Errorf("%w: logs of all pods are offered for %v", engine.ErrNotApplicable, engine.WorkloadLogKinds)
			}
			if _, err := streamable(s, "logs"); err != nil {
				return err
			}
			_, err := s.Lookup(p.Ref)
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p refParams) (any, error) {
			tab, created, err := s.OpenWorkloadLogsTab(ctx, p.Ref)
			if err != nil {
				return nil, err
			}
			return DockResult{Tab: tab, Created: created}, nil
		},
	})

	add(g, engine.Spec[logOptionsParams]{
		ID: "logs.setOptions", Label: "Log Options", Danger: engine.DangerUI,
		Description: "Changes a logs tab's options, as the tab's toolbar does: previous (the previous terminated " +
			"container's log), timestamps, resourceNames, wrap, search and filter. Options left out keep their value. " +
			"Changing previous or timestamps opens the stream again; the others change how the tab shows its lines. " +
			"ui_state lists every tab's options. Returns the tab.",
		Enabled: func(s *engine.Session, p logOptionsParams) error {
			_, err := s.LogsTab(p.ID)
			return err
		},
		Run: func(_ context.Context, s *engine.Session, p logOptionsParams) (any, error) {
			if p.Search != nil && len(*p.Search) > engine.MaxLogSearch {
				return nil, fmt.Errorf("search is %d bytes; at most %d", len(*p.Search), engine.MaxLogSearch)
			}
			return s.SetLogOptions(p.ID, func(o *engine.LogOptions) {
				set := func(dst *bool, v *bool) {
					if v != nil {
						*dst = *v
					}
				}
				set(&o.Previous, p.Previous)
				set(&o.Timestamps, p.Timestamps)
				set(&o.ResourceNames, p.ResourceNames)
				set(&o.Wrap, p.Wrap)
				set(&o.Filter, p.Filter)
				if p.Search != nil {
					o.Search = *p.Search
				}
			})
		},
	})

	add(g, engine.Spec[dockParams]{
		ID: "logs.controller", Label: "Logs of the Controlling Resource", Danger: engine.DangerRead, Verbs: workloadLogVerbs,
		Description: "Turns a pod's logs tab into the logs of the resource that controls the pod (its Deployment, " +
			"Daemon Set, Stateful Set, Replica Set or Job; a Deployment's pods through their replica set), as " +
			"workload_logs shows them: every pod's containers, each line with its pod and container. Returns the tab.",
		Enabled: func(s *engine.Session, p dockParams) error {
			tab, err := s.LogsTab(p.ID)
			if err != nil {
				return err
			}
			if tab.Ref.Kind != kindPods {
				return fmt.Errorf("%w: the tab shows %s %s already", engine.ErrNotApplicable, tab.Ref.Kind, tab.Ref.Name)
			}
			if _, err := streamable(s, "logs"); err != nil {
				return err
			}
			_, err = s.PodController(*tab.Ref)
			return err
		},
		Run: func(ctx context.Context, s *engine.Session, p dockParams) (any, error) {
			return s.MoveLogsTab(ctx, p.ID)
		},
	})
}
