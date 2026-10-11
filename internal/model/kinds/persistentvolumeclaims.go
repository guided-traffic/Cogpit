package kinds

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

const (
	pvcBound   = "Bound"
	pvcPending = "Pending"
	pvcLost    = "Lost"
	// pvcTerminating is the Status text of a claim with a deletion timestamp.
	pvcTerminating = "Terminating"
	// storageClassAnnotation is the class annotation that predates spec.storageClassName.
	storageClassAnnotation = "volume.beta.kubernetes.io/storage-class"
	resourceStorage        = "storage"
)

// persistentVolumeClaims has the columns of Lens's documentation (not captured). The Pods column
// (the pods that mount the claim) needs other objects than the claim, which a profile cannot see:
// it keeps its place and stays empty until a cross-kind index exists. Storage Class is plain text,
// not a link: Cogpit has no Storage Classes view yet, and a link would open an error.
var persistentVolumeClaims = &model.KindProfile{
	Key:        "persistentvolumeclaims",
	Version:    "v1",
	Resource:   "persistentvolumeclaims",
	Kind:       "PersistentVolumeClaim",
	Title:      "Persistent Volume Claims",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "storageClass", Title: "Storage Class", Type: model.ColumnText},
		{ID: "size", Title: "Size", Type: model.ColumnNumber}, // sorted by bytes
		{ID: idPods, Title: titlePods, Type: model.ColumnText},
		ageColumn,
		statusColumn,
	},
	Cells: pvcCells,
	Fields: map[string]fieldFunc{
		"storageClass": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(pvcStorageClass(obj))
		},
		"volumeName": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "spec", "volumeName"))
		},
		"accessMode": func(obj *unstructured.Unstructured) []string {
			return model.NestedStrings(obj, "spec", "accessModes")
		},
		"volumeMode": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "spec", "volumeMode"))
		},
		idStatus: func(obj *unstructured.Unstructured) []string {
			text, _ := pvcStatus(obj)
			return []string{text}
		},
	},
	Detail: pvcDetail,
}

func pvcCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	text, token := pvcStatus(obj)
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: pvcStorageClass(obj)},
		pvcSize(obj),
		{}, // Pods: needs a cross-kind index (the pods whose volumes name this claim)
		model.AgeCell(created(obj)),
		{Text: text, Status: token},
	}, token
}

// pvcStatus is status.phase: Bound ok, Pending pending, Lost error (the volume is gone), and
// Terminating once deleted, as kubectl prints it: a claim that a pod still uses stays in that
// state (pvc-protection), which is worth seeing. The API server sets a phase on every claim, so a
// missing one is shown as Unknown rather than guessed.
func pvcStatus(obj any) (string, model.Status) {
	if deleting(obj) {
		return pvcTerminating, model.StatusTerminating
	}
	switch phase := model.NestedString(obj, "status", "phase"); phase {
	case pvcBound:
		return phase, model.StatusOK
	case pvcPending:
		return phase, model.StatusPending
	case pvcLost:
		return phase, model.StatusError
	case "":
		return textUnknown, model.StatusUnknown
	default:
		return phase, model.StatusUnknown
	}
}

// pvcStorageClass is spec.storageClassName, else the older annotation when the field is not set.
// An explicit "" means no class: the claim binds only to volumes without one.
func pvcStorageClass(obj any) string {
	if _, set := model.NestedValue(obj, "spec", "storageClassName"); set {
		return model.NestedString(obj, "spec", "storageClassName")
	}
	return model.NestedString(obj, "metadata", "annotations", storageClassAnnotation)
}

// pvcSize is the claim's capacity once bound (status.capacity), else what it requests. Num is the
// size in bytes, so the column sorts by size.
func pvcSize(obj any) model.Cell {
	text := model.NestedIntOrString(obj, "status", "capacity", resourceStorage)
	if text == "" {
		text = model.NestedIntOrString(obj, "spec", "resources", "requests", resourceStorage)
	}
	if text == "" {
		return model.Cell{Text: "-"}
	}
	return model.Cell{Text: text, Num: quantityValue(text)}
}

// quantityValue returns a quantity's value as a sort key, or 0 when it does not parse. It skips
// decimal exponents beyond ±30, which no size uses and which resource.ParseQuantity spends
// unbounded time on (it rounds a tiny value at full precision); objects from the API server are
// canonical, but the cache's content is treated as hostile.
func quantityValue(s string) int64 {
	if s == "" || len(s) > 64 {
		return 0
	}
	if i := strings.IndexAny(s, "eE"); i >= 0 {
		if rest := s[i+1:]; rest != "" && rest != "i" { // "E" and "Ei" are suffixes
			exp, err := strconv.Atoi(rest)
			if err != nil || exp < -30 || exp > 30 {
				return 0
			}
		}
	}
	q, err := resource.ParseQuantity(s)
	if err != nil {
		return 0
	}
	return q.Value()
}

func pvcDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	text, token := pvcStatus(obj)
	d.addStatus(titleStatus, text, token)
	class := pvcStorageClass(obj)
	if _, set := model.NestedValue(obj, "spec", "storageClassName"); set && class == "" {
		class = "none (binds only to volumes without a class)"
	}
	d.add("Storage Class", class)
	d.add("Volume", model.NestedString(obj, "spec", "volumeName"))
	d.add("Access Modes", strings.Join(model.NestedStrings(obj, "spec", "accessModes"), ", "))
	d.add("Volume Mode", model.NestedString(obj, "spec", "volumeMode"))
	d.add("Requested", model.NestedIntOrString(obj, "spec", "resources", "requests", resourceStorage))
	d.add("Limit", model.NestedIntOrString(obj, "spec", "resources", "limits", resourceStorage))
	d.add("Capacity", model.NestedIntOrString(obj, "status", "capacity", resourceStorage))
	d.add("Allocated", model.NestedIntOrString(obj, "status", "allocatedResources", resourceStorage))
	resize := model.NestedString(obj, "status", "allocatedResourceStatuses", resourceStorage)
	d.addStatus("Resize Status", resize, resizeToken(resize))
	d.add("Volume Attributes Class", model.NestedString(obj, "spec", "volumeAttributesClassName"))
	d.add("Data Source", pvcDataSource(obj))
	d.add("Selector", strings.Join(labelSelector(obj, "spec", "selector"), "\n"))
	sections := withSection(nil, "Persistent Volume Claim", d)

	if cs, ok := conditionSection(conditions(obj, "status", "conditions"), pvcConditionToken); ok {
		sections = append(sections, cs)
	}
	return sections
}

// pvcDataSource renders the claim's data source (dataSourceRef, else dataSource) as "Kind/name",
// with its namespace when it is in another one: "VolumeSnapshot/nightly, namespace backups".
func pvcDataSource(obj any) string {
	src := model.NestedMap(obj, "spec", "dataSourceRef")
	if src == nil {
		src = model.NestedMap(obj, "spec", "dataSource")
	}
	kind, name := model.NestedString(src, "kind"), model.NestedString(src, "name")
	if name == "" {
		return ""
	}
	s := name
	if kind != "" {
		s = kind + "/" + name
	}
	if ns := model.NestedString(src, "namespace"); ns != "" {
		s += ", namespace " + ns
	}
	return s
}

// resizeToken colours status.allocatedResourceStatuses: a resize in progress or waiting for the
// node is pending, an infeasible one an error.
func resizeToken(s string) model.Status {
	switch {
	case s == "":
		return model.StatusNone
	case strings.HasSuffix(s, "Infeasible"):
		return model.StatusError
	default:
		return model.StatusPending
	}
}

// pvcConditionToken colours the claim's conditions. All of them report a change in flight
// (Resizing, FileSystemResizePending, ModifyingVolume) or its failure (ControllerResizeError,
// NodeResizeError, ModifyVolumeError) when True.
func pvcConditionToken(c condition) model.Status {
	whenTrue := model.StatusPending
	if strings.HasSuffix(c.Type, "Error") {
		whenTrue = model.StatusError
	}
	return conditionToken(c.Status, whenTrue, model.StatusNone)
}
