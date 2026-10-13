package engine

import (
	"github.com/guided-traffic/cogpit/internal/model/kinds"
)

// Argo CD (G23): the curated profiles of its three kinds come through the custom kinds watch like
// any curated profile (kinds.ProfileFor). Its settings, Repositories and Clusters, are virtual
// kinds over the Secrets Argo CD labels (kinds/argocd_settings.go); no definition announces them,
// so they come with Argo CD's Application definition: offered while the cluster serves
// applications.argoproj.io and the connection serves Secrets, gone with the definition, as
// Lens's documentation says ("whenever Argo CD is installed"; Lens finds the definitions at
// connect, Cogpit follows them live).

// addArgoCompanions adds the settings views to a build of the custom kinds when it holds Argo CD's
// Application kind and the backend offers Secrets. The profiles are shared and never change, so a
// rebuild keeps their pointers and the open view keeps running.
func addArgoCompanions(b *Backend, next map[string]customEntry) {
	app, ok := next[kinds.ArgoApplicationsKey]
	if !ok || b.builtinProfile(secretsKind) == nil {
		return
	}
	for _, p := range kinds.ArgoSettings() {
		if _, taken := next[p.Key]; !taken && b.builtinProfile(p.Key) == nil {
			next[p.Key] = customEntry{kind: app.kind, profile: p}
		}
	}
}
