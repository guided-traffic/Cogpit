package engine

import (
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/alerting"
	"github.com/guided-traffic/cogpit/internal/model"
)

// The alert board (D-77; the owner, 2026-10-17: "ein Alert-Dashboard … wie karma, aber
// aufgeräumter"): a workspace page of the shown cluster's active alerts, grouped by severity
// (critical, warning, info), a card per alertname with its count, and per alert its summary (the
// description folded under it), since when it fires, the object it names as a link into Cogpit's
// views and its runbook. It shows what serves and nothing else ("sparsam bei der Darstellung"): no
// label chips, no cluster, environment, job, instance or Prometheus labels, no receivers, no
// silences. Read only: silencing is a write to Alertmanager, not Cogpit's (D-77). The model is
// alerts.board's; agents read the alerts flat with alerts.list.

// AlertBoardModel is the board: the cluster's alerts as the window shows them, with its search and
// namespace filter applied.
type AlertBoardModel struct {
	Context string `json:"context"`
	// State, Where and Detail are what Cogpit knows of the Alertmanager (AlertsSummary's).
	State  AlertsState `json:"state"`
	Where  string      `json:"where,omitempty"`
	Detail string      `json:"detail,omitempty"`
	// AsOf is when the alerts shown were answered; FailingSince when the Alertmanager stopped
	// answering (the alerts are then its last answer's).
	AsOf         time.Time `json:"asOf,omitzero"`
	FailingSince time.Time `json:"failingSince,omitzero"`
	// Total counts every shown alert (the pill's number), Matched those the search and the
	// namespace filter keep; Truncated says Alertmanager answered more than Cogpit keeps.
	Total     int  `json:"total"`
	Matched   int  `json:"matched"`
	Truncated bool `json:"truncated,omitempty"`
	// Filter, FilterError and Namespaces are the board's search and namespace filter (the UI
	// state's).
	Filter      string   `json:"filter,omitempty"`
	FilterError string   `json:"filterError,omitempty"`
	Namespaces  []string `json:"namespaces,omitempty"`
	// Groups are the severities that have alerts the filters keep, worst first.
	Groups []AlertGroup `json:"groups"`
}

// AlertGroup is one severity of the board.
type AlertGroup struct {
	// Key is the group's fold key and its severity: critical, warning, info.
	Key   string `json:"key"`
	Title string `json:"title"` // "Critical"
	// Status is the severity's colour token: error, warning, pending (Solarized red, orange, yellow).
	Status model.Status `json:"status"`
	Count  int          `json:"count"`
	Folded bool         `json:"folded,omitempty"`
	Cards  []AlertCard  `json:"cards"`
}

// AlertCard is one alertname within a severity.
type AlertCard struct {
	// Key is the card's fold key: "<severity>/<alertname>".
	Key    string      `json:"key"`
	Name   string      `json:"alertname"`
	Count  int         `json:"count"`
	Folded bool        `json:"folded,omitempty"`
	Alerts []AlertItem `json:"alerts"`
}

// AlertItem is one active alert as the board and alerts.list show it.
type AlertItem struct {
	// Fingerprint is Alertmanager's id of the alert; the key of its description's fold.
	Fingerprint string `json:"fingerprint"`
	Name        string `json:"alertname"`
	// Severity is how it is shown (critical, warning, info); SeverityLabel the label as the rule
	// wrote it, when it says something else ("major", or none: "").
	Severity      string `json:"severity"`
	SeverityLabel string `json:"severityLabel,omitempty"`
	// Summary and Description are the rule's annotations, as written: text from the cluster.
	Summary     string `json:"summary,omitempty"`
	Description string `json:"description,omitempty"`
	// Expanded says the board shows the description (alerts.fold).
	Expanded bool `json:"expanded,omitempty"`
	// Since is when it started firing.
	Since time.Time `json:"since"`
	// Objects are what its labels name: the namespace, then the object the alert is about, as
	// links into Cogpit's views; a container as text with its pod.
	Objects []AlertObject `json:"objects,omitempty"`
	// Runbook is the runbook_url annotation when it is an http or https URL; others are left out.
	Runbook string `json:"runbook,omitempty"`
}

// AlertObject is an object an alert's labels name.
type AlertObject struct {
	// Label is the label that names it: namespace, pod, deployment, node, …; "customresource" for
	// kube-state-metrics' custom resource labels.
	Label string `json:"label"`
	// Ref is the object; Ref.Kind is empty when the cluster serves no kind Cogpit shows for it
	// (the window shows the name as text).
	Ref model.ObjectRef `json:"ref"`
	// Title is the kind's title, singular as the board writes it: "Deployment", "Kustomization".
	Title string `json:"title"`
	// Container is a pod's container the alert names; empty otherwise.
	Container string `json:"container,omitempty"`
}

// AlertList is alerts.list's answer: the shown alerts, flat, with the Alertmanager's state.
type AlertList struct {
	Context      string      `json:"context"`
	State        AlertsState `json:"state"`
	Where        string      `json:"where,omitempty"`
	Detail       string      `json:"detail,omitempty"`
	AsOf         time.Time   `json:"asOf,omitzero"`
	FailingSince time.Time   `json:"failingSince,omitzero"`
	// Total counts every shown alert; Critical, Warning and Info by severity (the pill's).
	Total     int  `json:"total"`
	Critical  int  `json:"critical"`
	Warning   int  `json:"warning"`
	Info      int  `json:"info"`
	Truncated bool `json:"truncated,omitempty"`
	// Matched counts the alerts the parameters keep; Alerts are at most the limit of them.
	Matched int         `json:"matched"`
	Alerts  []AlertItem `json:"alerts"`
}

// alertSeverities are the board's groups: severity, title, colour.
var alertSeverities = []struct {
	sev    alerting.Severity
	status model.Status
}{
	{alerting.SeverityCritical, model.StatusError},
	{alerting.SeverityWarning, model.StatusWarning},
	{alerting.SeverityInfo, model.StatusPending},
}

// severityTitle is a severity's group title: "Critical".
func severityTitle(s alerting.Severity) string {
	return strings.ToUpper(string(s[:1])) + string(s[1:])
}

// AlertSeverityStatus is a severity's colour token: critical red, warning orange, info yellow (the
// owner, 2026-10-17); StatusNone for anything else.
func AlertSeverityStatus(severity string) model.Status {
	for _, s := range alertSeverities {
		if string(s.sev) == severity {
			return s.status
		}
	}
	return model.StatusNone
}

// AlertBoard returns the board of a connected cluster ("" the shown one): its alerts with the UI
// state's search and namespace filter applied, grouped and folded as the board shows them. It sends
// no request.
func (s *Session) AlertBoard(contextID string) (AlertBoardModel, error) {
	b, snap, err := s.alertSnapshot(contextID)
	if err != nil {
		return AlertBoardModel{}, err
	}
	u := s.UIState()
	text := u.Filter
	if !u.FilterOptions.Regex {
		text = strings.TrimSpace(text)
	}
	m, ferr := NewRowMatcher(text, u.FilterOptions)
	out := AlertBoardModel{Context: b.Context, State: AlertsState(snap.State), Detail: snap.Error, AsOf: snap.AsOf,
		FailingSince: snap.FailingSince, Total: snap.Total, Truncated: snap.Truncated, Filter: u.Filter,
		Namespaces: slices.Clone(u.Namespaces), Groups: []AlertGroup{}}
	if ferr != nil {
		out.FilterError = ferr.Error()
	}
	if snap.Target != (alerting.Target{}) {
		out.Where = snap.Target.Where()
	}
	items := s.alertItems(b, snap.Alerts, u.AlertBoard)
	folded := func(key string) bool { return slices.Contains(u.AlertBoard.Folded, key) }
	for _, sv := range alertSeverities {
		g := AlertGroup{Key: string(sv.sev), Title: severityTitle(sv.sev), Status: sv.status, Folded: folded(string(sv.sev))}
		for _, it := range items {
			if it.Severity != string(sv.sev) || !alertInNamespaces(it, u.Namespaces) || !m.Match(alertRow(it)) {
				continue
			}
			key := it.Severity + "/" + it.Name
			if n := len(g.Cards); n == 0 || g.Cards[n-1].Key != key {
				g.Cards = append(g.Cards, AlertCard{Key: key, Name: it.Name, Folded: folded(key)})
			}
			c := &g.Cards[len(g.Cards)-1]
			c.Alerts = append(c.Alerts, it)
			c.Count++
			g.Count++
		}
		if g.Count > 0 {
			out.Matched += g.Count
			out.Groups = append(out.Groups, g)
		}
	}
	return out, nil
}

// AlertListParams narrow alerts.list.
type AlertListParams struct {
	Severities []string // critical, warning, info; none: all
	Filter     string   // terms each alert must contain (alertname, summary, description, objects)
	Namespace  string   // only alerts in this namespace
	Limit      int      // at most this many alerts; 0: 200
}

// maxAlertList bounds alerts.list's answer: an agent asks for more with a narrower filter.
const maxAlertList = 1000

// AlertList returns a connected cluster's shown alerts ("" the shown one), flat, sorted as the board
// lists them, narrowed by p. It sends no request.
func (s *Session) AlertList(contextID string, p AlertListParams) (AlertList, error) {
	b, snap, err := s.alertSnapshot(contextID)
	if err != nil {
		return AlertList{}, err
	}
	sum := summaryOf(snap)
	out := AlertList{Context: b.Context, State: sum.State, Where: sum.Where, Detail: sum.Detail, AsOf: snap.AsOf,
		FailingSince: snap.FailingSince, Total: sum.Total, Critical: sum.Critical, Warning: sum.Warning, Info: sum.Info,
		Truncated: snap.Truncated, Alerts: []AlertItem{}}
	limit := p.Limit
	if limit <= 0 {
		limit = 200
	}
	limit = min(limit, maxAlertList)
	m, _ := NewRowMatcher(strings.TrimSpace(p.Filter), FilterOptions{})
	for _, it := range s.alertItems(b, snap.Alerts, AlertBoardState{}) {
		if len(p.Severities) > 0 && !slices.Contains(p.Severities, it.Severity) {
			continue
		}
		if p.Namespace != "" && !alertInNamespaces(it, []string{p.Namespace}) {
			continue
		}
		if !m.Match(alertRow(it)) {
			continue
		}
		out.Matched++
		if len(out.Alerts) < limit {
			out.Alerts = append(out.Alerts, it)
		}
	}
	return out, nil
}

// alertItems turns a snapshot's alerts into the board's items, sorted as the board lists them.
func (s *Session) alertItems(b *Backend, alerts []alerting.Alert, fold AlertBoardState) []AlertItem {
	sorted := slices.Clone(alerts)
	alerting.SortAlerts(sorted)
	kinds := alertKinds(b)
	out := make([]AlertItem, 0, len(sorted))
	for _, a := range sorted {
		it := AlertItem{Fingerprint: a.Fingerprint, Name: a.Name, Severity: string(a.Severity), Summary: a.Summary,
			Description: a.Description, Since: a.StartsAt, Objects: alertObjects(a.Labels, kinds), Runbook: webURL(a.Runbook),
			Expanded: slices.Contains(fold.Expanded, a.Fingerprint)}
		if !strings.EqualFold(a.SeverityLabel, string(a.Severity)) {
			it.SeverityLabel = a.SeverityLabel
		}
		if it.Summary == "" && it.Description != "" { // a rule with a description only: its first line
			it.Summary, _, _ = strings.Cut(it.Description, "\n")
		}
		out = append(out, it)
	}
	return out
}

// alertRow is an alert as the search sees it: its alertname, severity, summary, description and
// the names of its objects.
func alertRow(it AlertItem) model.Row {
	cells := make([]model.Cell, 0, 5+3*len(it.Objects))
	cells = append(cells, model.Cell{Text: it.Name}, model.Cell{Text: it.Severity}, model.Cell{Text: it.SeverityLabel},
		model.Cell{Text: it.Summary}, model.Cell{Text: it.Description})
	for _, o := range it.Objects {
		cells = append(cells, model.Cell{Text: o.Ref.Namespace}, model.Cell{Text: o.Ref.Name}, model.Cell{Text: o.Container})
	}
	return model.Row{Cells: cells}
}

// alertInNamespaces reports whether an alert lies in one of the namespaces (all without any): its
// namespace object's. An alert that names no namespace (a node's, the cluster's) is in none.
func alertInNamespaces(it AlertItem, namespaces []string) bool {
	if len(namespaces) == 0 {
		return true
	}
	for _, o := range it.Objects {
		if o.Label == namespaceKey {
			return slices.Contains(namespaces, o.Ref.Name)
		}
	}
	return false
}

// webURL returns u when it is an absolute http or https URL with a host, else "": a runbook link
// comes from the cluster, and the window opens it in the browser.
func webURL(u string) string {
	p, err := url.Parse(u)
	if err != nil || (p.Scheme != "http" && p.Scheme != "https") || p.Host == "" || p.User != nil {
		return ""
	}
	return u
}

// alertSubjects are the labels that name the object an alert is about, most specific first, as
// kube-state-metrics and the kube-prometheus rules write them. The first one an alert has is its
// subject. Not among them: service, endpoint, instance and job, which name the scraped target
// (often the exporter, not the object); container alone.
var alertSubjects = []struct {
	label, kind string // the kind key; empty: the label's plural, label + "s"
	cluster     bool   // a cluster-scoped kind: named without a namespace
}{
	{"deployment", "", false},
	{"statefulset", "", false},
	{"daemonset", "", false},
	{"replicaset", "", false},
	{"cronjob", "", false},
	{"job_name", "jobs", false},
	{"horizontalpodautoscaler", "", false},
	{"poddisruptionbudget", "", false},
	{"persistentvolumeclaim", "", false},
	{"ingress", "ingresses", false},
	{"persistentvolume", "", true},
	{"node", "", true},
}

// alertKindIndex finds a kind's key and title by the label's kind key, or by a custom resource's
// group and kind.
type alertKindIndex struct {
	byKey   map[string]*model.KindProfile
	byGroup map[string]*model.KindProfile // "<group>/<Kind>"
}

func alertKinds(b *Backend) alertKindIndex {
	ix := alertKindIndex{byKey: map[string]*model.KindProfile{}, byGroup: map[string]*model.KindProfile{}}
	for _, p := range b.AllProfiles() {
		ix.byKey[p.Key] = p
		if p.Group != "" && p.Kind != "" {
			ix.byGroup[p.Group+"/"+p.Kind] = p
		}
	}
	return ix
}

// object makes the AlertObject of a kind key and a name; a kind the cluster does not serve keeps its
// name only.
func (ix alertKindIndex) object(label, kind, namespace, name, title string) AlertObject {
	o := AlertObject{Label: label, Ref: model.ObjectRef{Namespace: namespace, Name: name}, Title: title}
	if p := ix.byKey[kind]; p != nil {
		o.Ref.Kind, o.Title = p.Key, alertKindTitle(p)
		if !p.Namespaced {
			o.Ref.Namespace = ""
		}
	}
	return o
}

// alertKindTitle is a kind's name as the board writes it beside an object: its API kind
// ("Deployment"), else its title.
func alertKindTitle(p *model.KindProfile) string {
	if p.Kind != "" {
		return p.Kind
	}
	return p.Title
}

// alertObjects reads the objects an alert's labels name: the namespace, then its subject, the most
// specific object (alertSubjects; a custom resource by kube-state-metrics' customresource labels;
// a Secret by x509-certificate-exporter's secret labels; a pod with its container). A pod is the
// subject only when nothing more specific is named and it is not the scraped target's own pod (its
// name begins with the job's or the service's name, as an exporter's does): a workload's alert
// carries the exporter's pod beside the workload (D-77).
func alertObjects(labels map[string]string, ix alertKindIndex) []AlertObject {
	ns := labels[namespaceKey]
	subject := alertSubject(labels, ix, ns)
	if subject != nil && subject.Ref.Namespace != "" {
		ns = subject.Ref.Namespace // the object's own namespace, where the label names another (secret_namespace)
	}
	var out []AlertObject
	if ns != "" {
		out = append(out, ix.object(namespaceKey, "namespaces", "", ns, "Namespace"))
	}
	if subject != nil {
		out = append(out, *subject)
	}
	return out
}

func alertSubject(labels map[string]string, ix alertKindIndex, ns string) *AlertObject {
	if kind, name := labels["customresource_kind"], labels["name"]; kind != "" && name != "" {
		o := AlertObject{Label: "customresource", Ref: model.ObjectRef{Namespace: ns, Name: name}, Title: kind}
		if p := ix.byGroup[labels["customresource_group"]+"/"+kind]; p != nil {
			o.Ref.Kind = p.Key
			if !p.Namespaced {
				o.Ref.Namespace = ""
			}
		}
		return &o
	}
	for _, sub := range alertSubjects {
		if name := labels[sub.label]; name != "" {
			if ns == "" && !sub.cluster {
				continue // a namespaced kind without a namespace names nothing Cogpit can find
			}
			kind := sub.kind
			if kind == "" {
				kind = sub.label + "s"
			}
			o := ix.object(sub.label, kind, ns, name, alertTitle(sub.label))
			return &o
		}
	}
	if name, sns := labels["secret_name"], labels["secret_namespace"]; name != "" && sns != "" {
		o := ix.object("secret", "secrets", sns, name, "Secret")
		return &o
	}
	if pod := labels["pod"]; pod != "" && ns != "" && !scrapedPod(pod, labels) {
		o := ix.object("pod", "pods", ns, pod, "Pod")
		o.Container = labels["container"]
		return &o
	}
	return nil
}

// scrapedPod reports whether a pod label names the scraped target's pod (an exporter's): its name
// begins with the job's or the service's name and a dash.
func scrapedPod(pod string, labels map[string]string) bool {
	for _, k := range []string{"job", "service"} {
		if v := labels[k]; v != "" && strings.HasPrefix(pod, v+"-") {
			return true
		}
	}
	return false
}

// alertTitle is a subject label's kind as the board writes it while the cluster serves no kind for
// it: "Deployment".
func alertTitle(label string) string {
	switch label {
	case "job_name":
		return "Job"
	case "horizontalpodautoscaler":
		return "HorizontalPodAutoscaler"
	case "poddisruptionbudget":
		return "PodDisruptionBudget"
	case "persistentvolumeclaim":
		return "PersistentVolumeClaim"
	case "persistentvolume":
		return "PersistentVolume"
	case "statefulset":
		return "StatefulSet"
	case "daemonset":
		return "DaemonSet"
	case "replicaset":
		return "ReplicaSet"
	case "cronjob":
		return "CronJob"
	}
	return strings.ToUpper(label[:1]) + label[1:]
}
