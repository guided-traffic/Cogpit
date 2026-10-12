package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/guided-traffic/cogpit/internal/engine"
)

// Helm chart repositories (G11, architecture §8): the public https repositories the user adds,
// kept in Cogpit's own configuration, their indexes in Cogpit's own cache. Adding and removing one
// changes no cluster, but it decides where charts come from and which https servers Cogpit reaches
// from the user's computer: like the kubeconfig sources, it is the user's own choice, so an
// agent's add or remove is refused before anyone is asked (the lead's default, 2026-10-13; the GUI
// and the CLI run it). An update downloads the indexes of repositories the user added already, and
// the list reads them: both run for every origin.

type repoAddParams struct {
	Name string `json:"name" jsonschema:"the repository's name: 1 to 63 letters, digits, '-' or '_' (no dot), e.g. bitnami"`
	URL  string `json:"url" jsonschema:"the repository's https URL, the folder its index.yaml is in, e.g. https://charts.example.com/stable; public repositories only (no user, password or query)"`
}

type repoNameParams struct {
	Name string `json:"name" jsonschema:"the repository's name as helm_repo_list lists it"`
}

type repoUpdateParams struct {
	Names []string `json:"names,omitempty" jsonschema:"the repositories to update; empty: every repository"`
}

// helmRepos returns the session's repositories, or why there are none.
func helmRepos(s *engine.Session) (*engine.HelmRepos, error) {
	if h := s.HelmRepos(); h != nil {
		return h, nil
	}
	return nil, engine.ErrNoHelmRepos
}

// errReposAreTheUsers refuses an agent's change of the repositories.
var errReposAreTheUsers = errors.New("the Helm repositories are the user's to add and remove, in Cogpit's window or its CLI; " +
	"agents may list them, update their indexes and install their charts (with the user's acceptance)")

// refuseAgents refuses an agent's request before anyone is asked.
func refuseAgents[P any](ctx context.Context, _ *engine.Session, _ P) error {
	if engine.OriginOf(ctx).IsAgent() {
		return errReposAreTheUsers
	}
	return nil
}

func registerHelmRepos(g *registrar) {
	add(g, engine.Spec[struct{}]{
		ID: "helm.repo.list", Label: "Helm Repositories", Danger: engine.DangerRead,
		Description: "Lists the Helm chart repositories added to Cogpit, in the order they were added: name, URL, when " +
			"each was added and its index last downloaded, how many charts the index lists, and the last download's " +
			"error. Their charts are the helmcharts kind (resources_list with kind helmcharts).",
		Enabled: func(s *engine.Session, _ struct{}) error { _, err := helmRepos(s); return err },
		Run: func(_ context.Context, s *engine.Session, _ struct{}) (any, error) {
			h, err := helmRepos(s)
			if err != nil {
				return nil, err
			}
			return h.List()
		},
	})

	add(g, engine.Spec[repoAddParams]{
		ID: "helm.repo.add", Label: "Add Helm Repository", Danger: engine.DangerUI, Refuse: refuseAgents[repoAddParams],
		Description: "Adds a public Helm chart repository by name and https URL: Cogpit downloads its index.yaml (at most " +
			"32 MiB) and keeps the repository in its own configuration, not in Helm's. Changes no cluster. No credentials: " +
			"a repository that needs a login, a client certificate or plain http cannot be added. Refused for agents: " +
			"the user adds repositories.",
		Enabled: func(s *engine.Session, _ repoAddParams) error { _, err := helmRepos(s); return err },
		Run: func(ctx context.Context, s *engine.Session, p repoAddParams) (any, error) {
			h, err := helmRepos(s)
			if err != nil {
				return nil, err
			}
			info, err := h.Add(ctx, p.Name, p.URL)
			if err != nil {
				return nil, err
			}
			userToast(s, engine.LevelInfo, fmt.Sprintf("Helm repository %s added: %d charts", info.Name, info.Charts))
			return info, nil
		},
	})

	add(g, engine.Spec[repoNameParams]{
		ID: "helm.repo.remove", Label: "Remove Helm Repository", Danger: engine.DangerUI, Refuse: refuseAgents[repoNameParams],
		Description: "Removes a Helm chart repository from Cogpit and deletes its downloaded index; its charts leave the " +
			"helmcharts kind. Releases installed from it stay as they are. Changes no cluster. Refused for agents.",
		Enabled: func(s *engine.Session, _ repoNameParams) error { _, err := helmRepos(s); return err },
		Run: func(_ context.Context, s *engine.Session, p repoNameParams) (any, error) {
			h, err := helmRepos(s)
			if err != nil {
				return nil, err
			}
			if err := h.Remove(p.Name); err != nil {
				return nil, err
			}
			userToast(s, engine.LevelInfo, "Helm repository "+p.Name+" removed")
			return h.List()
		},
	})

	add(g, engine.Spec[repoUpdateParams]{
		ID: "helm.repo.update", Label: "Update Helm Repositories", Danger: engine.DangerUI,
		Description: "Downloads the index of every Helm repository added to Cogpit (or of those named) again, one after the " +
			"other, as helm repo update does. A repository whose download fails keeps the index it had and reports the " +
			"error. Changes no cluster. While the Charts view or a Helm Install tab is open, Cogpit does this itself for " +
			"an index older than 10 minutes.",
		Enabled: func(s *engine.Session, _ repoUpdateParams) error { _, err := helmRepos(s); return err },
		Run: func(ctx context.Context, s *engine.Session, p repoUpdateParams) (any, error) {
			h, err := helmRepos(s)
			if err != nil {
				return nil, err
			}
			infos, err := h.Update(ctx, p.Names)
			if err != nil {
				if infos == nil {
					return nil, err
				}
				userToast(s, engine.LevelWarning, "Not every Helm repository was updated: "+err.Error())
				return infos, nil
			}
			if len(infos) == 1 {
				userToast(s, engine.LevelInfo, "Helm repository "+infos[0].Name+" updated")
			} else {
				userToast(s, engine.LevelInfo, fmt.Sprintf("%d Helm repositories updated", len(infos)))
			}
			return infos, nil
		},
	})
}
