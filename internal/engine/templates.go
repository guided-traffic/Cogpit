package engine

import (
	"embed"
	"fmt"
	"path"
	"slices"
	"strings"
	"sync"
)

// The Create resource tab's templates are Cogpit's own (clean-room, conventions): one manifest per
// kind in templates/, without a namespace, so that a create lands in the namespace the user has
// chosen. They are checked against a real API server with a dry run in the kind tests.
//
//go:embed templates/*.yaml
var templateFS embed.FS

// TemplateGroup is the heading the template picker shows above Cogpit's templates.
const TemplateGroup = "Cogpit"

// DefaultTemplate is the template a Create resource tab starts with.
const DefaultTemplate = "Pod"

// Template is one manifest to start a Create resource tab from.
type Template struct {
	Name       string `json:"name"` // the kind it creates, "Deployment"
	Group      string `json:"group"`
	APIVersion string `json:"apiVersion"`
	Namespaced bool   `json:"namespaced"`
	Text       string `json:"yaml"`
}

var (
	templatesOnce sync.Once
	templates     []Template
	templatesErr  error
)

// Templates returns Cogpit's templates in alphabetical order of their kinds.
func Templates() ([]Template, error) {
	templatesOnce.Do(func() {
		entries, err := templateFS.ReadDir("templates")
		if err != nil {
			templatesErr = err
			return
		}
		for _, e := range entries {
			b, err := templateFS.ReadFile(path.Join("templates", e.Name()))
			if err != nil {
				templatesErr = err
				return
			}
			obj, err := ParseManifest(string(b))
			if err != nil {
				templatesErr = fmt.Errorf("template %s: %w", e.Name(), err)
				return
			}
			templates = append(templates, Template{
				Name: obj.GetKind(), Group: TemplateGroup, APIVersion: obj.GetAPIVersion(),
				Namespaced: !clusterScopedTemplate[obj.GetKind()], Text: string(b),
			})
		}
		slices.SortFunc(templates, func(a, b Template) int { return strings.Compare(a.Name, b.Name) })
	})
	return slices.Clone(templates), templatesErr
}

// clusterScopedTemplate names the template kinds that live outside namespaces, for the picker's
// hint; the create itself asks discovery.
var clusterScopedTemplate = map[string]bool{
	"ClusterRole": true, "ClusterRoleBinding": true, "CustomResourceDefinition": true,
	"DeviceClass": true, "IngressClass": true, "Namespace": true, "PersistentVolume": true,
	"PriorityClass": true, "StorageClass": true,
}

// TemplateByName returns the template for a kind, case-insensitively ("deployment").
func TemplateByName(name string) (Template, error) {
	all, err := Templates()
	if err != nil {
		return Template{}, err
	}
	for _, t := range all {
		if strings.EqualFold(t.Name, strings.TrimSpace(name)) {
			return t, nil
		}
	}
	names := make([]string, 0, len(all))
	for _, t := range all {
		names = append(names, t.Name)
	}
	return Template{}, fmt.Errorf("no template %q; templates: %s", name, strings.Join(names, ", "))
}
