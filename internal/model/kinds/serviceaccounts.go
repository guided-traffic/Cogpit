package kinds

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The fields of a service account that name Secrets.
const (
	fieldSecrets          = "secrets"
	fieldImagePullSecrets = "imagePullSecrets"
)

// serviceAccounts has the columns of Lens's documentation (not captured): Name, Namespace, Age. A
// service account has no state, so no status token. The drawer is Cogpit's (inferred): whether
// pods mount its token, and its Secrets and image pull secrets by name, never their content.
var serviceAccounts = &model.KindProfile{
	Key:        keyServiceAccounts,
	Version:    "v1",
	Resource:   keyServiceAccounts,
	Kind:       kindServiceAccount,
	Title:      "Service Accounts",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		ageColumn,
	},
	Cells: serviceAccountCells,
	Fields: map[string]fieldFunc{
		"secret": func(obj *unstructured.Unstructured) []string {
			return nameList(obj, fieldSecrets)
		},
		"imagePullSecret": func(obj *unstructured.Unstructured) []string {
			return nameList(obj, fieldImagePullSecrets)
		},
	},
	Detail: serviceAccountDetail,
}

func serviceAccountCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// serviceAccountDetail adds to Properties: Automount Token (the API's default, true, when the
// account does not say), then its Secrets and Image Pull Secrets as chips that link to them.
func serviceAccountDetail(obj *unstructured.Unstructured) []model.DetailSection {
	ns := model.NestedString(obj, "metadata", "namespace")
	var props detailItems
	automount := "true (default)"
	if v, ok := model.NestedBool(obj, "automountServiceAccountToken"); ok {
		automount = "false"
		if v {
			automount = textTrue
		}
	}
	props.add("Automount Token", automount)
	for _, list := range []struct{ item, field string }{{"Secrets", fieldSecrets}, {"Image Pull Secrets", fieldImagePullSecrets}} {
		names := nameList(obj, list.field)
		chips := make([]model.Chip, 0, len(names))
		for _, n := range names {
			chips = append(chips, model.Chip{Text: n, Link: linkTo(keySecrets, ns, n)})
		}
		props.put(model.ChipsItem(list.item, chips))
	}
	return []model.DetailSection{{Title: model.TitleProperties, Items: props}}
}
