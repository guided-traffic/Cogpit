package engine

// HelmInstallTab is what a Helm Install tab shows beside its values editor (inferred: Lens's
// "Helm Install: <chart>" tab is not captured): the chart and the repository it comes from, the
// version the install takes and the versions its picker offers, whether the archive matched its
// index's digest, and the namespace and release name the tab starts with. The tab's Ref is the
// chart (kind helmcharts); its values are the tab's editor text (editor.get, editor.setText),
// which agents read withheld (model.Withheld), as a Helm Upgrade tab's.
type HelmInstallTab struct {
	Repository string `json:"repository"`
	Chart      string `json:"chart"`
	Version    string `json:"version"`
	AppVersion string `json:"appVersion,omitempty"`
	// Versions are the picker's choices: every version the repository's index lists, newest first.
	Versions []string `json:"versions"`
	// Digest is the downloaded archive's SHA-256; Verified says it matched the index's digest
	// (false: the index lists none).
	Digest   string `json:"digest"`
	Verified bool   `json:"verified"`
	// Namespace and ReleaseName are what the tab offers at first: the namespace filter's single
	// namespace (else "default") and a name made from the chart's.
	Namespace   string `json:"namespace"`
	ReleaseName string `json:"releaseName"`
}
