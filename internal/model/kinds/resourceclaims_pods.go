package kinds

import (
	"strings"

	"github.com/guided-traffic/cogpit/internal/model"
)

// titleContainers heads the column of the containers that request a claim.
const titleContainers = "Containers"

// podResourceClaimsSection is the pod drawer's Resource Claims section, shown when the pod uses a
// claim (Lens's documentation, read 2026-10-13: each claim, whether the pod references it directly
// or through a template, its state, and the containers that request it). Built: each of the pod's
// claims with the claim it resolves to (named directly, or generated from a template and recorded
// in status.resourceClaimStatuses), a link to it and to the template, and the containers and init
// containers that request it ("app" or "app (gpu)" for one request of the claim). Not built: the
// claim's state, which would make the Pods view hold a watch on every claim. The section's place,
// after Pod Volumes, is inferred.
func podResourceClaimsSection(pod any, namespace string) (model.DetailSection, bool) {
	claims := model.NestedSlice(pod, "spec", "resourceClaims")
	if len(claims) == 0 {
		return model.DetailSection{}, false
	}
	generated := map[string]string{} // the pod's claim name → the generated claim
	for _, s := range model.NestedSlice(pod, "status", "resourceClaimStatuses") {
		generated[model.NestedString(s, fieldName)] = model.NestedString(s, "resourceClaimName")
	}
	cols := []model.Column{
		{Title: titleName, Type: model.ColumnText}, {Title: titleClaim, Type: model.ColumnText},
		{Title: "Template", Type: model.ColumnText}, {Title: titleContainers, Type: model.ColumnText},
	}
	rows := make([][]model.Cell, 0, len(claims))
	for _, c := range claims {
		name := model.NestedString(c, fieldName)
		if name == "" {
			continue
		}
		claim := model.NestedString(c, "resourceClaimName")
		template := model.NestedString(c, "resourceClaimTemplateName")
		claimCell := model.Cell{Text: claim, Link: linkTo(keyResourceClaims, namespace, claim)}
		if claim == "" && template != "" {
			if g := generated[name]; g != "" {
				claimCell = model.Cell{Text: g, Link: linkTo(keyResourceClaims, namespace, g)}
			} else {
				claimCell = model.Cell{Text: "(not generated yet)"}
			}
		}
		rows = append(rows, []model.Cell{
			{Text: name},
			claimCell,
			{Text: template, Link: linkTo(keyResourceClaimTemplates, namespace, template)},
			{Text: strings.Join(claimContainers(pod, name), "\n")},
		})
	}
	if len(rows) == 0 {
		return model.DetailSection{}, false
	}
	return model.DetailSection{Title: "Resource Claims", Table: model.CellTable(cols, rows)}, true
}

// claimContainers names the init containers and containers whose resources.claims use the pod's
// claim: "app", or "app (gpu)" when the container asks for one request of it.
func claimContainers(pod any, claim string) []string {
	var out []string
	for _, field := range []string{fieldInitContainers, fieldContainers} {
		for _, c := range model.NestedSlice(pod, "spec", field) {
			for _, rc := range model.NestedSlice(c, "resources", "claims") {
				if model.NestedString(rc, fieldName) != claim {
					continue
				}
				text := model.NestedString(c, fieldName)
				if req := model.NestedString(rc, "request"); req != "" {
					text += " (" + req + ")"
				}
				out = appendUnique(out, text)
			}
		}
	}
	return out
}
