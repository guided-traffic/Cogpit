package kinds

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Applications (resource-parity › Cluster › Applications): Lens's documentation (read 2026-10-13,
// not captured) describes an application as "a set of various Kubernetes resources that have labels
// binding them to the application instance" and points at Kubernetes' recommended labels. Cogpit
// groups the cached workloads (Deployments, Stateful Sets, Daemon Sets, Cron Jobs, Jobs) by their
// app.kubernetes.io/instance label, per namespace. The objects are Cogpit's own (apiVersion
// cogpit.io/v1, kind Application), made by GroupApplications from the workloads; the engine keeps
// them current from the cache and serves them as a virtual kind (architecture §5).

// The recommended labels Applications reads.
const (
	LabelAppInstance  = "app.kubernetes.io/instance"
	LabelAppName      = "app.kubernetes.io/name"
	LabelAppVersion   = "app.kubernetes.io/version"
	LabelAppManagedBy = "app.kubernetes.io/managed-by"
	LabelAppPartOf    = "app.kubernetes.io/part-of"
	LabelAppComponent = "app.kubernetes.io/component"
)

// ApplicationsKey is the kind key of the Applications view: Cogpit's own bare plural, like
// helmreleases (architecture §4.5). No Kubernetes API serves it; Argo CD's applications are
// applications.argoproj.io.
const ApplicationsKey = "applications"

// ApplicationWorkloadKinds are the kind keys of the workloads applications are made of, in the
// order the drawer lists them. A Job a Cron Job controls is left out: the Cron Job stands for it.
var ApplicationWorkloadKinds = []string{keyDeployments, keyStatefulSets, keyDaemonSets, keyCronJobs, keyJobs}

// Kind keys and API kinds of the workloads, and titles and fields of the view.
const (
	keyDeployments  = "deployments"
	keyStatefulSets = "statefulsets"
	keyDaemonSets   = "daemonsets"
	keyCronJobs     = "cronjobs"
	keyJobs         = "jobs"
	kindDeployment  = "Deployment"
	kindCronJob     = "CronJob"
	kindJob         = "Job"
	titleName       = "Name"
	titleVersion    = "Version"
	fieldLabels     = "labels"
)

// The texts of an application's Status, by the worst status among its workloads (inferred).
const (
	appHealthy     = "Healthy"
	appProgressing = "Progressing"
	appDegraded    = "Degraded"
	appFailing     = "Failing"
	appTerminating = "Terminating"
	appUnknown     = "Unknown"
	appIdle        = "Idle"
	idViolations   = "violations"
	idWorkloads    = "workloads"
	idManagedBy    = "managedBy"
	idAppVersion   = "version"
	idAppName      = "appName"
)

// AppWorkload is one workload as GroupApplications reads it: its kind key, the object, and its
// row's status as the kind's view colours it.
type AppWorkload struct {
	Kind   string
	Object *unstructured.Unstructured
	Status model.Status
}

// Applications returns the profile of the Applications view. It is virtual: its objects are made
// from other kinds' cached objects, so Edit and Delete do not apply and its drawer is the profile's
// alone. Derive passes the objects through: the engine hands them over grouped already.
func Applications() *model.KindProfile { return applications }

var applications = &model.KindProfile{
	Key:        ApplicationsKey,
	Group:      model.LocalGroup,
	Version:    "v1",
	Resource:   ApplicationsKey,
	Kind:       "Application",
	Title:      "Applications",
	Namespaced: true,
	Columns: []model.Column{
		{ID: fieldName, Title: "Instance", Type: model.ColumnText},
		namespaceColumn,
		{ID: idManagedBy, Title: "Managed by", Type: model.ColumnText, Width: 120},
		{ID: idAppVersion, Title: titleVersion, Type: model.ColumnText, Width: 110},
		{ID: idViolations, Title: "Violations", Type: model.ColumnNumber},
		ageColumn,
		statusColumn,
		{ID: idAppName, Title: titleName, Type: model.ColumnText, Hidden: true},
		{ID: idWorkloads, Title: "Workloads", Type: model.ColumnNumber, Hidden: true},
	},
	Cells: applicationCells,
	Fields: map[string]fieldFunc{
		idManagedBy:  func(obj *unstructured.Unstructured) []string { return model.NestedStrings(obj, "spec", idManagedBy) },
		idAppVersion: func(obj *unstructured.Unstructured) []string { return model.NestedStrings(obj, "spec", "versions") },
		idAppName:    func(obj *unstructured.Unstructured) []string { return model.NestedStrings(obj, "spec", "names") },
		"workload": func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, w := range model.NestedSlice(obj, "spec", idWorkloads) {
				out = appendUnique(out, model.NestedString(w, fieldName))
			}
			return out
		},
		idStatus: func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "status", "text"))
		},
	},
	Derive: func(objs []*unstructured.Unstructured) []*unstructured.Unstructured { return objs },
	Detail: applicationDetail,
}

func applicationCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	status := model.Status(model.NestedString(obj, "status", idStatus))
	violations, _ := model.NestedInt64(obj, "status", idViolations)
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: strings.Join(model.NestedStrings(obj, "spec", idManagedBy), ", ")},
		{Text: strings.Join(model.NestedStrings(obj, "spec", "versions"), ", ")},
		{Num: max(violations, 0)},
		model.AgeCell(created(obj)),
		{Text: model.NestedString(obj, "status", "text"), Status: status},
		{Text: strings.Join(model.NestedStrings(obj, "spec", "names"), ", ")},
		{Num: int64(len(model.NestedSlice(obj, "spec", idWorkloads)))},
	}, status
}

func applicationDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.add("Created", timestamp(created(obj)))
	d.add("Instance", obj.GetName())
	d.put(linkedItem("Namespace", obj.GetNamespace(), linkTo(keyNamespaces, "", obj.GetNamespace())))
	d.add(titleName, strings.Join(model.NestedStrings(obj, "spec", "names"), ", "))
	d.add("Part Of", strings.Join(model.NestedStrings(obj, "spec", "partOf"), ", "))
	d.add("Components", strings.Join(model.NestedStrings(obj, "spec", "components"), ", "))
	d.add("Managed by", strings.Join(model.NestedStrings(obj, "spec", idManagedBy), ", "))
	d.add(titleVersion, strings.Join(model.NestedStrings(obj, "spec", "versions"), ", "))
	d.addStatus(titleStatus, model.NestedString(obj, "status", "text"), model.Status(model.NestedString(obj, "status", idStatus)))
	if n, ok := model.NestedInt64(obj, "status", idViolations); ok {
		d.add("Violations", itoa(max(n, 0)))
	}
	sections := withSection(nil, model.TitleProperties, d)

	list := model.NestedSlice(obj, "spec", idWorkloads)
	if len(list) == 0 {
		return sections
	}
	cols := []model.Column{
		{Title: titleKind, Type: model.ColumnText}, {Title: titleName, Type: model.ColumnText},
		{Title: titleVersion, Type: model.ColumnText}, {Title: titleStatus, Type: model.ColumnText},
	}
	rows := make([][]model.Cell, 0, len(list))
	for _, w := range list {
		status := model.Status(model.NestedString(w, idStatus))
		name := model.NestedString(w, fieldName)
		rows = append(rows, []model.Cell{
			{Text: model.NestedString(w, idKind)},
			{Text: name, Link: linkTo(model.NestedString(w, "key"), obj.GetNamespace(), name)},
			{Text: model.NestedString(w, idAppVersion)},
			{Text: appStatusText(status), Status: status},
		})
	}
	return append(sections, model.DetailSection{Title: "Workloads", Table: model.CellTable(cols, rows)})
}

// ApplicationWorkloads returns references to an application's workloads, in the drawer's order.
func ApplicationWorkloads(app *unstructured.Unstructured) []model.ObjectRef {
	list := model.NestedSlice(app, "spec", idWorkloads)
	out := make([]model.ObjectRef, 0, len(list))
	for _, w := range list {
		key, name := model.NestedString(w, "key"), model.NestedString(w, fieldName)
		if slices.Contains(ApplicationWorkloadKinds, key) && name != "" {
			out = append(out, model.ObjectRef{Kind: key, Namespace: app.GetNamespace(), Name: name})
		}
	}
	return out
}

// appStatusRank orders the workloads' statuses: the highest wins an application's Status.
var appStatusRank = map[model.Status]int{
	model.StatusError: 7, model.StatusWarning: 6, model.StatusPending: 5, model.StatusTerminating: 4,
	model.StatusUnknown: 3, model.StatusOK: 2, model.StatusSucceeded: 2, model.StatusStopped: 1,
}

// appStatusText names a status the way the Applications view's Status column does (inferred:
// Lens's documentation lists no values).
func appStatusText(s model.Status) string {
	switch s {
	case model.StatusError:
		return appFailing
	case model.StatusWarning:
		return appDegraded
	case model.StatusPending:
		return appProgressing
	case model.StatusTerminating:
		return appTerminating
	case model.StatusUnknown:
		return appUnknown
	case model.StatusOK, model.StatusSucceeded:
		return appHealthy
	}
	return appIdle
}

// kindTitles names the workload kinds in the drawer.
var kindTitles = map[string]string{
	keyDeployments: kindDeployment, keyStatefulSets: "StatefulSet", keyDaemonSets: "DaemonSet",
	keyCronJobs: kindCronJob, keyJobs: kindJob,
}

// appGroup collects one application's workloads.
type appGroup struct {
	namespace, instance string
	created             time.Time
	names, partOf       []string
	components          []string
	managedBy, versions []string
	workloads           []appMember
}

type appMember struct {
	kind, name, version string
	status              model.Status
	order               int
}

// GroupApplications groups workloads into applications by namespace and app.kubernetes.io/instance
// and returns one object per application, sorted by namespace and instance. A workload without the
// label, or a Job a Cron Job controls, belongs to none. An application's Managed by, Version, Name,
// Part Of and Components list the distinct values of its workloads' labels, sorted; Age is its
// oldest workload's; Status is its worst workload's (Failing, Degraded, Progressing, Terminating,
// Unknown, Healthy, or Idle when no workload reports a state, a Deployment scaled to zero); and
// Violations counts the workloads in a warning or error state (inferred: Lens's documentation says
// only "information related to violations of the application functioning"). Pure: it modifies no
// input and tolerates any content.
func GroupApplications(ws []AppWorkload) []*unstructured.Unstructured {
	groups := map[string]*appGroup{}
	var keys []string
	for _, w := range ws {
		obj := w.Object
		if obj == nil {
			continue
		}
		order := slices.Index(ApplicationWorkloadKinds, w.Kind)
		if order < 0 || (w.Kind == keyJobs && controlledByCronJob(obj)) {
			continue
		}
		labels := model.NestedStringMap(obj, "metadata", "labels")
		instance, ns, name := labels[LabelAppInstance], model.NestedString(obj, "metadata", "namespace"), model.NestedString(obj, "metadata", fieldName)
		if instance == "" || ns == "" || name == "" || !appInstanceName(instance) {
			continue
		}
		key := ns + "/" + instance
		g := groups[key]
		if g == nil {
			g = &appGroup{namespace: ns, instance: instance}
			groups[key] = g
			keys = append(keys, key)
		}
		if t := created(obj); !t.IsZero() && (g.created.IsZero() || t.Before(g.created)) {
			g.created = t
		}
		g.names = appendUnique(g.names, labels[LabelAppName])
		g.partOf = appendUnique(g.partOf, labels[LabelAppPartOf])
		g.components = appendUnique(g.components, labels[LabelAppComponent])
		g.managedBy = appendUnique(g.managedBy, labels[LabelAppManagedBy])
		g.versions = appendUnique(g.versions, labels[LabelAppVersion])
		g.workloads = append(g.workloads, appMember{kind: w.Kind, name: name, version: labels[LabelAppVersion], status: w.Status, order: order})
	}
	slices.Sort(keys)
	out := make([]*unstructured.Unstructured, 0, len(keys))
	for _, k := range keys {
		out = append(out, groups[k].object())
	}
	return out
}

// appInstanceName reports whether an instance label can name an object: a label value is at most
// 63 characters of [A-Za-z0-9._-], so only an absurd one is refused; the object's name is the
// value as it is.
func appInstanceName(s string) bool {
	return len(s) <= 63 && !strings.ContainsAny(s, "/ \t\r\n")
}

// controlledByCronJob reports whether a Job is a Cron Job's run.
func controlledByCronJob(job *unstructured.Unstructured) bool {
	for _, o := range model.NestedSlice(job, "metadata", "ownerReferences") {
		if model.NestedString(o, idKind) == kindCronJob {
			if c, ok := model.NestedBool(o, "controller"); ok && c {
				return true
			}
		}
	}
	return false
}

func (g *appGroup) object() *unstructured.Unstructured {
	slices.SortFunc(g.workloads, func(a, b appMember) int {
		if a.order != b.order {
			return a.order - b.order
		}
		return strings.Compare(a.name, b.name)
	})
	for _, list := range [][]string{g.names, g.partOf, g.components, g.managedBy, g.versions} {
		slices.Sort(list)
	}
	worst, violations := model.StatusNone, int64(0)
	workloads := make([]any, 0, len(g.workloads))
	for _, w := range g.workloads {
		if appStatusRank[w.status] > appStatusRank[worst] {
			worst = w.status
		}
		if w.status == model.StatusWarning || w.status == model.StatusError {
			violations++
		}
		workloads = append(workloads, map[string]any{
			"key": w.kind, idKind: kindTitles[w.kind], fieldName: w.name, idAppVersion: w.version, idStatus: string(w.status),
		})
	}
	if worst == model.StatusSucceeded {
		worst = model.StatusOK // a finished Job is part of a healthy application
	}
	meta := map[string]any{
		fieldName: g.instance, "namespace": g.namespace, "uid": appUID(g.namespace, g.instance),
		fieldLabels: map[string]any{LabelAppInstance: g.instance},
	}
	if !g.created.IsZero() {
		meta["creationTimestamp"] = g.created.UTC().Format(time.RFC3339)
	}
	return &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": model.LocalGroup + "/v1",
		"kind":       "Application",
		"metadata":   meta,
		"spec": map[string]any{
			"names": anyStrings(g.names), "partOf": anyStrings(g.partOf), "components": anyStrings(g.components),
			idManagedBy: anyStrings(g.managedBy), "versions": anyStrings(g.versions), idWorkloads: workloads,
		},
		"status": map[string]any{idStatus: string(worst), "text": appStatusText(worst), idViolations: violations},
	}}
}

// appUID is an application's UID: the same for the same namespace and instance, so that its row and
// the drawer's binding survive every rebuild.
func appUID(namespace, instance string) string {
	sum := sha256.Sum256([]byte("application/" + namespace + "/" + instance))
	return hex.EncodeToString(sum[:16])
}

func anyStrings(list []string) []any {
	out := make([]any, len(list))
	for i, s := range list {
		out[i] = s
	}
	return out
}
