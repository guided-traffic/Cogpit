package commands

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Logs of an application: its label (Lens's) and the most tabs it opens.
const (
	applicationLogsLabel  = "Logs"
	maxApplicationLogTabs = 8
)

// ApplicationLogsResult is what application.logs did: a logs tab per workload, and the workloads
// it left out (a Cron Job, whose Jobs have their own logs, or one past the bound).
type ApplicationLogsResult struct {
	Tabs    []DockResult      `json:"tabs"`
	Skipped []model.ObjectRef `json:"skipped,omitempty"`
}

// errNoLoggableWorkload refuses Logs on an application without a workload workload.logs serves.
var errNoLoggableWorkload = fmt.Errorf("%w: the application has no workload whose logs a tab shows (%v)", engine.ErrNotApplicable, engine.WorkloadLogKinds)

func registerApplications(g *registrar) {
	// Lens's documentation (read 2026-10-13) lists Logs as the application's one action: "Displays
	// the application log". Lens's own form is not captured; Cogpit opens workload.logs' tab for each
	// of the application's workloads.
	add(g, engine.Spec[refParams]{
		ID: "application.logs", Label: applicationLogsLabel, MenuPath: "Application/Logs", Kinds: []string{kinds.ApplicationsKey}, Order: 300,
		Danger: engine.DangerRead, Verbs: workloadLogVerbs,
		Description: "Opens the logs of an application (applications view): a workload_logs dock tab for each of its " +
			"Deployments, Stateful Sets, Daemon Sets and Jobs, at most 8, each following every container of the " +
			"workload's pods. A Cron Job is left out. Activates tabs already open. Returns the tabs and the workloads " +
			"left out.",
		Enabled: func(s *engine.Session, p refParams) error {
			if p.Ref.Kind != kinds.ApplicationsKey {
				return fmt.Errorf("%w: Logs of an application need an applications reference", engine.ErrNotApplicable)
			}
			if _, err := streamable(s, "logs"); err != nil {
				return err
			}
			app, err := s.Lookup(p.Ref)
			if err != nil {
				return err
			}
			if !slices.ContainsFunc(kinds.ApplicationWorkloads(app), loggable) {
				return errNoLoggableWorkload
			}
			return nil
		},
		Run: func(ctx context.Context, s *engine.Session, p refParams) (any, error) {
			app, err := s.Object(ctx, p.Ref)
			if err != nil {
				return nil, err
			}
			res := ApplicationLogsResult{Tabs: []DockResult{}}
			var errs []error
			for _, w := range kinds.ApplicationWorkloads(app) {
				if !loggable(w) || len(res.Tabs) == maxApplicationLogTabs {
					res.Skipped = append(res.Skipped, w)
					continue
				}
				tab, created, err := s.OpenWorkloadLogsTab(ctx, w)
				if err != nil {
					errs = append(errs, fmt.Errorf("%s %s: %w", w.Kind, w.Name, err))
					continue
				}
				res.Tabs = append(res.Tabs, DockResult{Tab: tab, Created: created})
			}
			if len(res.Tabs) == 0 {
				if len(errs) == 0 {
					return nil, errNoLoggableWorkload
				}
				return nil, errors.Join(errs...)
			}
			return res, nil
		},
	})
}

// loggable reports whether workload.logs serves a workload's kind.
func loggable(w model.ObjectRef) bool { return slices.Contains(engine.WorkloadLogKinds, w.Kind) }
