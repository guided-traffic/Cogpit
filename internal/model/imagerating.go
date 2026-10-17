package model

import (
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Image ratings (D-78; resource-parity › Pods, Cogpit's own, the Trivy integration): the worst
// severity among the known vulnerabilities of a container image, as Trivy Operator's
// VulnerabilityReport (aquasecurity.github.io/v1alpha1) of one workload's container sums them up in
// report.summary. The owner's "Image-Rating" (2026-10-17): every Pod shows the worst of its
// containers', the pod's drawer each image's.
//
// The lead's decisions (2026-10-17, D-78): the rating is a severity word, not a letter grade (the
// scanner's own vocabulary, which needs no legend); the worst found of Critical, High, Medium, Low,
// Unknown (vulnerabilities without a severity, ranked above Clean and below Low: worse than none,
// not known to be worse than a low one), Clean (a report without vulnerabilities), and Not scanned
// (no report). Its text carries the count of that severity, "High · 3", so that one Critical
// reads apart from forty; Clean and Not scanned carry none. Colours are the status tokens, in the
// scanner's own order: Critical the error red, High the warning orange, Medium the pending
// yellow, Low the succeeded blue, Unknown grey, Clean the OK green, Not scanned none.

// The labels Trivy Operator puts on a report (seen on Trivy Operator 0.35.0, chart 0.37.0, on kind,
// 2026-10-17): the workload it scanned, by kind, name and namespace, and the container. A name that
// is no valid label value (longer than 63 characters) goes into the annotation of the same key
// instead, the label then holding a hash under TrivyLabelResourceNameHash.
const (
	TrivyLabelResourceKind      = "trivy-operator.resource.kind"
	TrivyLabelResourceName      = "trivy-operator.resource.name"
	TrivyLabelResourceNameHash  = "trivy-operator.resource.name-hash"
	TrivyLabelResourceNamespace = "trivy-operator.resource.namespace"
	TrivyLabelContainerName     = "trivy-operator.container.name"
)

// ImageRating is the worst severity among an image's known vulnerabilities. The zero value is
// RatingNotScanned; a greater rating is a worse one.
type ImageRating int

// The ratings, from the best to the worst.
const (
	RatingNotScanned ImageRating = iota // no report of the image
	RatingClean                         // a report without vulnerabilities
	RatingUnknown                       // vulnerabilities, none with a known severity
	RatingLow
	RatingMedium
	RatingHigh
	RatingCritical
)

var ratingNames = [...]string{"notScanned", "clean", "unknown", "low", "medium", "high", "critical"}

var ratingTitles = [...]string{"Not scanned", "Clean", "Unknown", "Low", "Medium", "High", "Critical"}

var ratingStatus = [...]Status{StatusNone, StatusOK, StatusUnknown, StatusSucceeded, StatusPending, StatusWarning, StatusError}

func (r ImageRating) valid() bool { return r >= RatingNotScanned && r <= RatingCritical }

// Name is the rating's name in JSON and for agents: "critical", "notScanned".
func (r ImageRating) Name() string {
	if !r.valid() {
		return ratingNames[RatingNotScanned]
	}
	return ratingNames[r]
}

// String is the rating as the window shows it: "Critical", "Not scanned".
func (r ImageRating) String() string {
	if !r.valid() {
		return ratingTitles[RatingNotScanned]
	}
	return ratingTitles[r]
}

// Status is the status token the rating is drawn in.
func (r ImageRating) Status() Status {
	if !r.valid() {
		return StatusNone
	}
	return ratingStatus[r]
}

// MarshalText writes the rating's name.
func (r ImageRating) MarshalText() ([]byte, error) { return []byte(r.Name()), nil }

// UnmarshalText reads a rating's name; an unknown name reads as RatingNotScanned.
func (r *ImageRating) UnmarshalText(b []byte) error {
	*r = RatingNotScanned
	for i, n := range ratingNames {
		if n == string(b) {
			*r = ImageRating(i)
		}
	}
	return nil
}

// VulnerabilityCounts are the vulnerabilities of one image by severity (a report's summary).
type VulnerabilityCounts struct {
	Critical int64 `json:"critical"`
	High     int64 `json:"high"`
	Medium   int64 `json:"medium"`
	Low      int64 `json:"low"`
	Unknown  int64 `json:"unknown"`
}

// Rating is the worst severity the counts hold: RatingClean for none.
func (c VulnerabilityCounts) Rating() ImageRating {
	switch {
	case c.Critical > 0:
		return RatingCritical
	case c.High > 0:
		return RatingHigh
	case c.Medium > 0:
		return RatingMedium
	case c.Low > 0:
		return RatingLow
	case c.Unknown > 0:
		return RatingUnknown
	}
	return RatingClean
}

// Of returns the count of one severity; 0 for Clean and Not scanned.
func (c VulnerabilityCounts) Of(r ImageRating) int64 {
	switch r {
	case RatingCritical:
		return c.Critical
	case RatingHigh:
		return c.High
	case RatingMedium:
		return c.Medium
	case RatingLow:
		return c.Low
	case RatingUnknown:
		return c.Unknown
	}
	return 0
}

// Total is the number of vulnerabilities of every severity.
func (c VulnerabilityCounts) Total() int64 { return c.Critical + c.High + c.Medium + c.Low + c.Unknown }

// RatingText is a rating as the window writes it: the severity and how many of it, "High · 3";
// "Clean" and "Not scanned" alone.
func RatingText(r ImageRating, c VulnerabilityCounts) string {
	if n := c.Of(r); n > 0 {
		return r.String() + " · " + strconv.FormatInt(n, 10)
	}
	return r.String()
}

// Chips are the counts as chips, the worst severity first, each in its rating's colour, leaving
// out the severities without a vulnerability: "1 Critical", "14 High". None for clean counts.
func (c VulnerabilityCounts) Chips() []Chip {
	var out []Chip
	for r := RatingCritical; r >= RatingUnknown; r-- {
		if n := c.Of(r); n > 0 {
			out = append(out, Chip{Text: strconv.FormatInt(n, 10) + " " + r.String(), Status: r.Status()})
		}
	}
	return out
}

// VulnerabilityReport is what Cogpit reads of one Trivy Operator VulnerabilityReport: the workload
// and container it is of, the image it scanned, the counts and when. The vulnerabilities themselves
// are not read: the report's own view and YAML show them.
type VulnerabilityReport struct {
	// Namespace and Name name the report object.
	Namespace string
	Name      string
	// Kind, Workload and Container name what was scanned: the workload's kind ("ReplicaSet",
	// "StatefulSet", "CronJob", "Pod"), its name, and the container's name in its pod template.
	Kind      string
	Workload  string
	Container string
	// Registry, Repository, Tag and Digest are the image scanned: "index.docker.io",
	// "library/nginx", "1.16.1", "sha256:…".
	Registry   string
	Repository string
	Tag        string
	Digest     string
	Counts     VulnerabilityCounts
	// Scanned is report.updateTimestamp, else the object's creation.
	Scanned time.Time
}

// ParseVulnerabilityReport reads a report object; ok is false for one that names no workload or
// container (not Trivy Operator's, or a shape it does not write).
func ParseVulnerabilityReport(obj *unstructured.Unstructured) (VulnerabilityReport, bool) {
	if obj == nil {
		return VulnerabilityReport{}, false
	}
	labels, ann := obj.GetLabels(), obj.GetAnnotations()
	r := VulnerabilityReport{
		Namespace: obj.GetNamespace(), Name: obj.GetName(),
		Kind: labels[TrivyLabelResourceKind], Workload: labels[TrivyLabelResourceName],
		Container: labels[TrivyLabelContainerName],
	}
	if r.Workload == "" {
		r.Workload = ann[TrivyLabelResourceName] // a name too long for a label
	}
	if r.Workload == "" { // the report's controller: the workload it was made for
		for _, o := range obj.GetOwnerReferences() {
			if o.Controller != nil && *o.Controller && o.Kind == r.Kind {
				r.Workload = o.Name
			}
		}
	}
	if ns := labels[TrivyLabelResourceNamespace]; ns != "" && r.Namespace == "" {
		r.Namespace = ns
	}
	if r.Kind == "" || r.Workload == "" || r.Container == "" {
		return VulnerabilityReport{}, false
	}
	report := NestedMap(obj.Object, "report")
	r.Registry = NestedString(report, "registry", "server")
	r.Repository = NestedString(report, "artifact", "repository")
	r.Tag = NestedString(report, "artifact", "tag")
	r.Digest = NestedString(report, "artifact", "digest")
	summary := NestedMap(report, "summary")
	r.Counts = VulnerabilityCounts{
		Critical: count(summary, "criticalCount"), High: count(summary, "highCount"),
		Medium: count(summary, "mediumCount"), Low: count(summary, "lowCount"),
		Unknown: count(summary, "unknownCount"),
	}
	if r.Scanned = NestedTime(report, "updateTimestamp"); r.Scanned.IsZero() {
		r.Scanned = obj.GetCreationTimestamp().Time
	}
	return r, true
}

// count reads a summary's count; a missing, negative or malformed one is 0.
func count(summary map[string]any, field string) int64 {
	n, _ := NestedInt64(summary, field)
	return max(n, 0)
}

// Image is the scanned image as one reference: "library/nginx:1.16.1", the repository and its tag,
// else its digest.
func (r VulnerabilityReport) Image() string {
	switch {
	case r.Tag != "":
		return r.Repository + ":" + r.Tag
	case r.Digest != "":
		return r.Repository + "@" + r.Digest
	}
	return r.Repository
}

// Matches reports whether the report can be of a container's image (its spec's image reference):
// the same repository, and the same tag when both name one. Trivy Operator replaces a report when
// the workload's pod template changes, so a report of another image is one not replaced yet: its
// rating is not the container's. The registry is not compared (Trivy writes Docker Hub as
// index.docker.io, a mirror as itself), nor the digest (an image index's and a platform's differ).
func (r VulnerabilityReport) Matches(image string) bool {
	repo, tag := ImageRepository(image)
	if repo == "" || r.Repository == "" {
		return true // nothing to compare
	}
	if repo != strings.TrimPrefix(r.Repository, "library/") {
		return false
	}
	return tag == "" || r.Tag == "" || tag == r.Tag
}

// ImageRepository splits a container's image reference into its repository, without the registry
// and without Docker Hub's "library/", and its tag: "nginx:1.16.1" and
// "docker.io/library/nginx:1.16.1@sha256:…" are ("nginx", "1.16.1"); "ghcr.io/org/app" is ("org/app",
// ""). An image without a tag or a digest is pulled as "latest".
func ImageRepository(image string) (repo, tag string) {
	image = strings.TrimSpace(image)
	digest := ""
	if i := strings.IndexByte(image, '@'); i >= 0 {
		image, digest = image[:i], image[i+1:]
	}
	if i := strings.LastIndexByte(image, ':'); i > strings.LastIndexByte(image, '/') {
		image, tag = image[:i], image[i+1:]
	}
	if i := strings.IndexByte(image, '/'); i >= 0 {
		if host := image[:i]; strings.ContainsAny(host, ".:") || host == "localhost" {
			image = image[i+1:]
		}
	}
	if tag == "" && digest == "" && image != "" {
		tag = "latest"
	}
	return strings.TrimPrefix(image, "library/"), tag
}
