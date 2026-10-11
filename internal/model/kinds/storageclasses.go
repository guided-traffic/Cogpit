package kinds

import (
	"sort"
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

const (
	groupStorage = "storage.k8s.io"
	// keyStorageClasses is the kind key class cells link to.
	keyStorageClasses = "storageclasses"
	// storageClassDefault marks the class that claims without a class get. Kubernetes counts only
	// the exact value "true"; the beta annotation is still honoured.
	storageClassDefault     = "storageclass.kubernetes.io/is-default-class"
	storageClassDefaultBeta = "storageclass.beta.kubernetes.io/is-default-class"
	// The API server's defaults for a class's reclaim policy and binding mode.
	reclaimDelete    = "Delete"
	bindingImmediate = "Immediate"
	idReclaimPolicy  = "reclaimPolicy"
	titleReclaim     = "Reclaim Policy"
)

// storageClasses has the columns of Lens's documentation (not captured): Name, Provisioner, Age,
// Reclaim Policy, Volume Binding Mode, in that order; the kind is cluster-scoped and the
// documentation lists no Namespace column. Which class is the default is not a documented column;
// the drawer and the searchable field "default" carry it (inferred). Storage classes have no state,
// so no status token.
var storageClasses = &model.KindProfile{
	Key:      keyStorageClasses,
	Group:    groupStorage,
	Version:  "v1",
	Resource: keyStorageClasses,
	Kind:     "StorageClass",
	Title:    "Storage Classes",
	Columns: []model.Column{
		nameColumn,
		{ID: "provisioner", Title: "Provisioner", Type: model.ColumnText},
		ageColumn,
		{ID: idReclaimPolicy, Title: titleReclaim, Type: model.ColumnText},
		{ID: "volumeBindingMode", Title: "Volume Binding Mode", Type: model.ColumnText},
	},
	Cells: storageClassCells,
	Fields: map[string]fieldFunc{
		"provisioner": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(model.NestedString(obj, "provisioner"))
		},
		idReclaimPolicy: func(obj *unstructured.Unstructured) []string {
			return []string{storageClassReclaim(obj)}
		},
		"volumeBindingMode": func(obj *unstructured.Unstructured) []string {
			return []string{storageClassBinding(obj)}
		},
		idDefault: func(obj *unstructured.Unstructured) []string {
			return []string{strconv.FormatBool(isDefaultStorageClass(obj))}
		},
	},
	Detail: storageClassDetail,
}

func storageClassCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		{Text: model.NestedString(obj, "provisioner")},
		model.AgeCell(created(obj)),
		{Text: storageClassReclaim(obj)},
		{Text: storageClassBinding(obj)},
	}, model.StatusNone
}

// storageClassReclaim is reclaimPolicy; the API server defaults it to Delete, so an object built
// without the default reads the same.
func storageClassReclaim(obj any) string {
	if s := model.NestedString(obj, "reclaimPolicy"); s != "" {
		return s
	}
	return reclaimDelete
}

// storageClassBinding is volumeBindingMode; the API server defaults it to Immediate.
func storageClassBinding(obj any) string {
	if s := model.NestedString(obj, "volumeBindingMode"); s != "" {
		return s
	}
	return bindingImmediate
}

func isDefaultStorageClass(obj any) bool {
	return model.NestedString(obj, "metadata", "annotations", storageClassDefault) == textTrue ||
		model.NestedString(obj, "metadata", "annotations", storageClassDefaultBeta) == textTrue
}

// storageClassDetail is Cogpit's drawer for a class (inferred, Lens's is not captured): the class
// in Properties, then its provisioner's parameters, one item each. Parameters name Secrets at most
// (csi.storage.k8s.io/provisioner-secret-name), never hold their values.
func storageClassDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	if isDefaultStorageClass(obj) {
		d.add("Default", "Yes: claims without a class get this one")
	}
	d.add("Provisioner", model.NestedString(obj, "provisioner"))
	d.add(titleReclaim, storageClassReclaim(obj))
	d.add("Volume Binding Mode", storageClassBinding(obj))
	if b, ok := model.NestedBool(obj, "allowVolumeExpansion"); ok {
		d.add("Allow Volume Expansion", strconv.FormatBool(b))
	}
	d.add("Mount Options", strings.Join(model.NestedStrings(obj, "mountOptions"), "\n"))
	d.add("Allowed Topologies", strings.Join(allowedTopologies(obj), "\n"))
	sections := withSection(nil, model.TitleProperties, d)

	var p detailItems
	params := model.NestedStringMap(obj, "parameters")
	for _, k := range sortedStringKeys(params) {
		p.add(k, params[k])
	}
	return withSection(sections, "Parameters", p)
}

// allowedTopologies renders each term of allowedTopologies as "key in (a,b)", its expressions
// joined by ", ": a volume may be provisioned in a topology that matches any one term.
func allowedTopologies(obj any) []string {
	var out []string
	for _, term := range model.NestedSlice(obj, "allowedTopologies") {
		var exprs []string
		for _, e := range model.NestedSlice(term, "matchLabelExpressions") {
			if key := model.NestedString(e, "key"); key != "" {
				exprs = append(exprs, key+" in ("+strings.Join(model.NestedStrings(e, "values"), ",")+")")
			}
		}
		if len(exprs) > 0 {
			out = append(out, strings.Join(exprs, ", "))
		}
	}
	return out
}

// sortedStringKeys returns a string map's keys in order.
func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
