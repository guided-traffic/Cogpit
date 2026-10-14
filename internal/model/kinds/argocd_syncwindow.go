package kinds

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// An AppProject's sync windows (A33, owner 2026-10-14 evening; H-67): a sync Cogpit writes is a
// manual sync, and it respects the windows as Argo CD's own check of a manual sync does
// (Argo CD's SyncWindows.CanSync, read from its documentation "Sync Windows" and its source,
// inferred, not captured against a running Argo CD): the windows that match the Application (its
// name, its destination namespace or cluster, by glob; with andOperator every list given must
// match) decide; an active deny window blocks unless it allows manual syncs (manualSync); else an
// active allow window lets it through; else, when allow windows match but none is active, it is
// blocked unless one of them allows manual syncs; without matching windows it goes through. A
// window is active from each time its cron schedule fires, in its time zone (UTC without one), for
// its duration. Argo CD's RBAC is not emulated. Pure: no I/O.

// The kinds of a sync window.
const (
	argoWindowAllow = "allow"
	argoWindowDeny  = "deny"
)

// ArgoWindow is one sync window of an AppProject as the check reads it.
type ArgoWindow struct {
	Kind         string // "allow" or "deny"
	Schedule     string
	Duration     time.Duration
	TimeZone     *time.Location
	Applications []string
	Namespaces   []string
	Clusters     []string
	ManualSync   bool
	AndOperator  bool
	sched        cron.Schedule
}

// argoWindowParser reads a window's schedule as Argo CD does: five fields or a descriptor.
var argoWindowParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)

// ArgoWindows reads an AppProject's spec.syncWindows. A window that cannot be read (its schedule,
// duration or time zone) is returned as an error: the caller does not guess.
func ArgoWindows(project *unstructured.Unstructured) ([]ArgoWindow, error) {
	if project == nil {
		return nil, nil
	}
	raw, _, _ := unstructured.NestedSlice(project.Object, "spec", "syncWindows")
	out := make([]ArgoWindow, 0, len(raw))
	for i, r := range raw {
		m, ok := r.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("sync window %d is not an object", i+1)
		}
		w := ArgoWindow{
			Kind: argoStr(m["kind"]), Schedule: argoStr(m["schedule"]), ManualSync: m["manualSync"] == true, AndOperator: m["andOperator"] == true,
			Applications: argoStrs(m["applications"]), Namespaces: argoStrs(m["namespaces"]), Clusters: argoStrs(m["clusters"]), TimeZone: time.UTC,
		}
		if w.Kind != argoWindowAllow && w.Kind != argoWindowDeny {
			return nil, fmt.Errorf("sync window %d: kind %q is neither allow nor deny", i+1, w.Kind)
		}
		var err error
		if w.sched, err = argoWindowParser.Parse(w.Schedule); err != nil {
			return nil, fmt.Errorf("sync window %d: schedule %q: %w", i+1, w.Schedule, err)
		}
		if w.Duration, err = time.ParseDuration(argoStr(m["duration"])); err != nil || w.Duration <= 0 {
			return nil, fmt.Errorf("sync window %d: duration %q is not a duration", i+1, argoStr(m["duration"]))
		}
		if tz := argoStr(m["timeZone"]); tz != "" {
			if w.TimeZone, err = time.LoadLocation(tz); err != nil {
				return nil, fmt.Errorf("sync window %d: time zone %q: %w", i+1, tz, err)
			}
		}
		out = append(out, w)
	}
	return out, nil
}

func argoStr(v any) string { s, _ := v.(string); return s }

func argoStrs(v any) []string {
	l, _ := v.([]any)
	out := make([]string, 0, len(l))
	for _, e := range l {
		if s, ok := e.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// Active reports whether the window is open at now, and until when.
func (w ArgoWindow) Active(now time.Time) (bool, time.Time) {
	local := now.In(w.TimeZone)
	start := w.sched.Next(local.Add(-w.Duration)) // the first firing after now-duration
	if start.After(local) {
		return false, time.Time{}
	}
	return true, start.Add(w.Duration)
}

// Matches reports whether the window applies to an Application named app with the destination
// namespace ns and cluster (its server URL or name).
func (w ArgoWindow) Matches(app, ns string, clusters []string) bool {
	lists := []struct {
		patterns []string
		values   []string
	}{{w.Applications, []string{app}}, {w.Namespaces, []string{ns}}, {w.Clusters, clusters}}
	given, matched := 0, 0
	for _, l := range lists {
		if len(l.patterns) == 0 {
			continue
		}
		given++
		if globAny(l.patterns, l.values) {
			matched++
		}
	}
	if w.AndOperator {
		return given > 0 && matched == given
	}
	return matched > 0
}

func globAny(patterns, values []string) bool {
	for _, p := range patterns {
		for _, v := range values {
			if v == "" {
				continue
			}
			if ok, _ := path.Match(p, v); ok {
				return true
			}
		}
	}
	return false
}

// String names the window for a message.
func (w ArgoWindow) String() string {
	return fmt.Sprintf("%s window %q for %s (%s)", w.Kind, w.Schedule, w.Duration, w.TimeZone)
}

// ArgoSyncBlocked returns why the AppProject's sync windows block a manual sync of app at now, ""
// when they do not; an error when a window cannot be read.
func ArgoSyncBlocked(app, project *unstructured.Unstructured, now time.Time) (string, error) {
	windows, err := ArgoWindows(project)
	if err != nil || len(windows) == 0 {
		return "", err
	}
	name := app.GetName()
	ns, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "namespace")
	server, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "server")
	dest, _, _ := unstructured.NestedString(app.Object, "spec", "destination", "name")
	clusters := []string{server, dest}
	var activeDeny, activeAllow, inactiveAllow []ArgoWindow
	var denyUntil time.Time
	for _, w := range windows {
		if !w.Matches(name, ns, clusters) {
			continue
		}
		on, until := w.Active(now)
		switch {
		case w.Kind == argoWindowDeny && on:
			activeDeny = append(activeDeny, w)
			if until.After(denyUntil) {
				denyUntil = until
			}
		case w.Kind == argoWindowAllow && on:
			activeAllow = append(activeAllow, w)
		case w.Kind == argoWindowAllow:
			inactiveAllow = append(inactiveAllow, w)
		}
	}
	if len(activeDeny) > 0 {
		for _, w := range activeDeny {
			if w.ManualSync {
				return "", nil
			}
		}
		return fmt.Sprintf("the AppProject's %s is open: Argo CD refuses syncs until %s", activeDeny[0],
			denyUntil.UTC().Format(time.RFC3339)), nil
	}
	if len(activeAllow) > 0 || len(inactiveAllow) == 0 {
		return "", nil
	}
	for _, w := range inactiveAllow {
		if w.ManualSync {
			return "", nil
		}
	}
	return fmt.Sprintf("no allow window of the AppProject is open (%s): Argo CD refuses syncs outside them",
		strings.TrimSpace(inactiveAllow[0].String())), nil
}
