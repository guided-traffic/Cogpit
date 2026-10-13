package kinds

import (
	"strconv"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

// Flux's image automation (image.toolkit.fluxcd.io) and notifications
// (notification.toolkit.fluxcd.io), the Flux section's Image Automation and Notifications
// sub-groups in the captured order. Columns and drawers inferred (flux.go).

// maxLatestTags bounds the tags an image repository's drawer lists (the scan keeps the newest ten).
const maxLatestTags = 10

// imageRepositoriesSpec: Image, Last Scan (an age) and Tags (how many the last scan found). v1
// since Flux 2.7; v1beta2 reads the same fields.
var imageRepositoriesSpec = &fluxSpec{
	group: fluxImage, resource: "imagerepositories", kind: "ImageRepository", title: "Image Repositories",
	version: "v1", fallback: []string{fluxV1beta2}, section: FluxSectionImages,
	suspend: true, reconcile: true,
	columns: []fluxColumn{
		{Column: model.Column{ID: "image", Title: "Image", Type: model.ColumnText}, cell: fluxTextCell("", fieldSpec, "image"), field: true},
		{Column: model.Column{ID: "lastScan", Title: "Last Scan", Type: model.ColumnAge, Width: 90}, cell: func(obj *unstructured.Unstructured) model.Cell {
			return model.AgeCell(model.NestedTime(obj, "status", "lastScanResult", "scanTime"))
		}},
		{Column: model.Column{ID: "tags", Title: "Tags", Type: model.ColumnNumber, Width: 70}, cell: func(obj *unstructured.Unstructured) model.Cell {
			n, ok := model.NestedInt64(obj, "status", "lastScanResult", "tagCount")
			if !ok {
				return model.Cell{Text: "-"}
			}
			return model.Cell{Text: itoa(n), Num: n}
		}},
	},
	properties: func(obj *unstructured.Unstructured, d *detailItems) {
		d.add("Image", model.NestedString(obj, fieldSpec, "image"))
		d.add("Provider", model.NestedString(obj, fieldSpec, "provider"))
		d.add("Insecure", fluxBool(obj, fieldSpec, "insecure"))
		d.put(fluxSecretItem(obj, titleSecret, fieldSpec, "secretRef"))
		d.put(fluxSecretItem(obj, "Cert Secret", fieldSpec, "certSecretRef"))
		d.add("Service Account", model.NestedString(obj, fieldSpec, "serviceAccountName"))
		d.add("Exclusion List", strings.Join(model.NestedStrings(obj, fieldSpec, "exclusionList"), "\n"))
		if n, ok := model.NestedInt64(obj, "status", "lastScanResult", "tagCount"); ok {
			d.add("Tags Found", itoa(n))
		}
		d.add("Last Scan", timestamp(model.NestedTime(obj, "status", "lastScanResult", "scanTime")))
		tags := model.NestedStrings(obj, "status", "lastScanResult", "latestTags")
		d.add("Latest Tags", strings.Join(tags[:min(len(tags), maxLatestTags)], "\n"))
	},
}

// imagePoliciesSpec: Image Repository (a link) and Latest Image (the image the policy selected:
// status.latestRef, v1beta2's status.latestImage). The image reflector computes policies from their
// repository's scans: Cogpit offers Reconcile, not Suspend. v1 since Flux 2.7; v1beta2 reads the
// same fields.
var imagePoliciesSpec = &fluxSpec{
	group: fluxImage, resource: "imagepolicies", kind: "ImagePolicy", title: "Image Policies",
	version: "v1", fallback: []string{fluxV1beta2}, section: FluxSectionImages,
	reconcile: true,
	columns: []fluxColumn{
		{Column: model.Column{ID: "imageRepository", Title: "Image Repository", Type: model.ColumnText}, cell: fluxRefCell("ImageRepository", fieldSpec, "imageRepositoryRef"), field: true},
		{Column: model.Column{ID: "latestImage", Title: "Latest Image", Type: model.ColumnText}, cell: func(obj *unstructured.Unstructured) model.Cell {
			return model.Cell{Text: latestImage(obj, "latestRef", "latestImage")}
		}, field: true},
	},
	properties: func(obj *unstructured.Unstructured, d *detailItems) {
		d.put(fluxRefItem(obj, "Image Repository", "ImageRepository", fieldSpec, "imageRepositoryRef"))
		d.add("Policy", imagePolicyRule(obj))
		d.add("Filter Tags", model.NestedString(obj, fieldSpec, "filterTags", "pattern"))
		d.add("Extract", model.NestedString(obj, fieldSpec, "filterTags", "extract"))
		d.add("Latest Image", latestImage(obj, "latestRef", "latestImage"))
		d.add("Previous Image", latestImage(obj, "observedPreviousRef", "observedPreviousImage"))
	},
}

// latestImage reads an image reference as v1 writes it ({name, tag, digest}) or as v1beta2 did (a
// string).
func latestImage(obj any, refField, imageField string) string {
	if ref := model.NestedMap(obj, "status", refField); ref != nil {
		s := model.NestedString(ref, fieldName)
		if tag := model.NestedString(ref, "tag"); tag != "" {
			s += ":" + tag
		}
		if digest := model.NestedString(ref, "digest"); digest != "" {
			s += "@" + digest
		}
		if s != "" {
			return s
		}
	}
	return model.NestedString(obj, "status", imageField)
}

// imagePolicyRule is the policy's rule: "semver 1.x", "alphabetical asc", "numerical desc".
func imagePolicyRule(obj any) string {
	policy := model.NestedMap(obj, fieldSpec, "policy")
	switch {
	case model.NestedMap(policy, "semver") != nil:
		return strings.TrimSpace("semver " + model.NestedString(policy, "semver", "range"))
	case model.NestedMap(policy, "alphabetical") != nil:
		return strings.TrimSpace("alphabetical " + model.NestedString(policy, "alphabetical", "order"))
	case model.NestedMap(policy, "numerical") != nil:
		return strings.TrimSpace("numerical " + model.NestedString(policy, "numerical", "order"))
	}
	return ""
}

// imageUpdateAutomationsSpec: Source (the Git repository it pushes to, a link) and Last Run (an
// age). v1 since Flux 2.7; v1beta2 reads the same fields.
var imageUpdateAutomationsSpec = &fluxSpec{
	group: fluxImage, resource: "imageupdateautomations", kind: "ImageUpdateAutomation", title: "Image Update Automations",
	version: "v1", fallback: []string{fluxV1beta2}, section: FluxSectionImages,
	suspend: true, reconcile: true,
	columns: []fluxColumn{
		{Column: model.Column{ID: idSource, Title: titleSource, Type: model.ColumnText}, cell: fluxRefCell("GitRepository", fieldSpec, "sourceRef"), field: true},
		{Column: model.Column{ID: "lastRun", Title: "Last Run", Type: model.ColumnAge, Width: 90}, cell: func(obj *unstructured.Unstructured) model.Cell {
			return model.AgeCell(model.NestedTime(obj, "status", "lastAutomationRunTime"))
		}},
	},
	properties: func(obj *unstructured.Unstructured, d *detailItems) {
		d.put(fluxRefItem(obj, titleSource, "GitRepository", fieldSpec, "sourceRef"))
		d.add("Checkout Branch", model.NestedString(obj, fieldSpec, "git", "checkout", idRef, "branch"))
		d.add("Push Branch", model.NestedString(obj, fieldSpec, "git", "push", "branch"))
		d.add("Update Path", model.NestedString(obj, fieldSpec, "update", "path"))
		d.add("Strategy", model.NestedString(obj, fieldSpec, "update", "strategy"))
		d.add("Last Run", timestamp(model.NestedTime(obj, "status", "lastAutomationRunTime")))
		d.add("Last Push Commit", model.NestedString(obj, "status", "lastPushCommit"))
		d.add("Last Push", timestamp(model.NestedTime(obj, "status", "lastPushTime")))
	},
}

// alertsSpec: Provider (a link) and Severity ("info" when unset, Flux's default). Notification
// v1beta3's Alerts are static objects without a status: ready unless suspended. A reconcile
// request does nothing to them, so Cogpit offers Reconcile only for Alerts that report a Ready
// condition (v1beta2's). v1beta3 since Flux 2.1; v1beta2 reads the same fields.
var alertsSpec = &fluxSpec{
	group: fluxNotification, resource: "alerts", kind: "Alert", title: "Alerts",
	version: "v1beta3", fallback: []string{fluxV1beta2}, section: FluxSectionNotifications,
	suspend: true, reconcile: true, static: true,
	columns: []fluxColumn{
		{Column: model.Column{ID: "provider", Title: "Provider", Type: model.ColumnText}, cell: fluxRefCell("Provider", fieldSpec, "providerRef"), field: true},
		{Column: model.Column{ID: "severity", Title: "Severity", Type: model.ColumnText, Width: 90}, cell: fluxTextCell("info", fieldSpec, "eventSeverity"), field: true},
	},
	properties: func(obj *unstructured.Unstructured, d *detailItems) {
		d.put(fluxRefItem(obj, "Provider", "Provider", fieldSpec, "providerRef"))
		d.add("Severity", fluxTextCell("info", fieldSpec, "eventSeverity")(obj).Text)
		d.put(model.ChipsItem("Event Sources", eventSourceChips(obj, model.NestedSlice(obj, fieldSpec, "eventSources"))))
		d.add("Inclusion List", strings.Join(model.NestedStrings(obj, fieldSpec, "inclusionList"), "\n"))
		d.add("Exclusion List", strings.Join(model.NestedStrings(obj, fieldSpec, "exclusionList"), "\n"))
		d.add("Summary", model.NestedString(obj, fieldSpec, "summary"))
	},
}

// eventSourceChips are an Alert's event sources or a Receiver's resources: "Kind/name" ("*" for
// every object of the kind), linked to the object when Cogpit knows its kind.
func eventSourceChips(obj *unstructured.Unstructured, list []any) []model.Chip {
	chips := make([]model.Chip, 0, len(list))
	for _, e := range list {
		kind, name := model.NestedString(e, idKind), model.NestedString(e, fieldName)
		if kind == "" && name == "" {
			continue
		}
		ns := model.NestedString(e, "namespace")
		text := name
		if ns != "" && ns != obj.GetNamespace() {
			text = ns + "/" + name
		} else {
			ns = obj.GetNamespace()
		}
		chip := model.Chip{Text: kind + "/" + text}
		if key := fluxRefKey(kind); key != "" && name != "*" && name != "" {
			chip.Link = linkTo(key, ns, name)
		}
		chips = append(chips, chip)
	}
	return chips
}

// providersSpec: Type (slack, github, generic, …) and Channel. A provider's address, proxy and
// credentials are never shown: an address can be a webhook URL with its token in it (spec.address,
// in the object's YAML, H-63). Static in notification v1beta3, as Alerts are.
var providersSpec = &fluxSpec{
	group: fluxNotification, resource: "providers", kind: "Provider", title: "Providers",
	version: "v1beta3", fallback: []string{fluxV1beta2}, section: FluxSectionNotifications,
	suspend: true, reconcile: true, static: true,
	columns: []fluxColumn{
		{Column: model.Column{ID: idType, Title: titleType, Type: model.ColumnText, Width: 110}, cell: fluxTextCell("", fieldSpec, idType), field: true},
		{Column: model.Column{ID: "channel", Title: "Channel", Type: model.ColumnText}, cell: fluxTextCell("", fieldSpec, "channel"), field: true},
	},
	properties: func(obj *unstructured.Unstructured, d *detailItems) {
		d.add(titleType, model.NestedString(obj, fieldSpec, idType))
		d.add("Channel", model.NestedString(obj, fieldSpec, "channel"))
		d.add("Username", model.NestedString(obj, fieldSpec, "username"))
		d.put(fluxSecretItem(obj, titleSecret, fieldSpec, "secretRef"))
		d.put(fluxSecretItem(obj, "Cert Secret", fieldSpec, "certSecretRef"))
	},
}

// receiversSpec: Type (github, gitlab, generic, …) and Resources (how many objects a webhook call
// reconciles). The webhook's path is never shown: for a generic receiver it is the only secret a
// caller needs (status.webhookPath, in the object's YAML, H-63). v1 since Flux 2.1; v1beta2 reads
// the same fields.
var receiversSpec = &fluxSpec{
	group: fluxNotification, resource: "receivers", kind: "Receiver", title: "Receivers",
	version: "v1", fallback: []string{fluxV1beta2}, section: FluxSectionNotifications,
	suspend: true, reconcile: true,
	columns: []fluxColumn{
		{Column: model.Column{ID: idType, Title: titleType, Type: model.ColumnText, Width: 110}, cell: fluxTextCell("", fieldSpec, idType), field: true},
		{Column: model.Column{ID: "resources", Title: titleResources, Type: model.ColumnNumber, Width: 90}, cell: func(obj *unstructured.Unstructured) model.Cell {
			n := int64(len(model.NestedSlice(obj, fieldSpec, "resources")))
			return model.Cell{Text: strconv.FormatInt(n, 10), Num: n}
		}},
	},
	properties: func(obj *unstructured.Unstructured, d *detailItems) {
		d.add(titleType, model.NestedString(obj, fieldSpec, idType))
		d.add("Events", strings.Join(model.NestedStrings(obj, fieldSpec, "events"), ", "))
		d.put(model.ChipsItem(titleResources, eventSourceChips(obj, model.NestedSlice(obj, fieldSpec, "resources"))))
		d.put(fluxSecretItem(obj, titleSecret, fieldSpec, "secretRef"))
	},
}
