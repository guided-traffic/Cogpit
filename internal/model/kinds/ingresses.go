package kinds

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/guided-traffic/cogpit/internal/model"
)

const (
	groupNetworking = "networking.k8s.io"
	// ingressClassAnnotation is the class annotation that predates spec.ingressClassName and that
	// controllers still honour.
	ingressClassAnnotation = "kubernetes.io/ingress.class"
	// anyHost stands for a rule without a host: it matches every host.
	anyHost = "*"
)

// ingresses has the columns of Lens's documentation (not captured). It has no status token: Lens
// shows none, and an Ingress without an address is not necessarily broken, because some
// controllers never publish one (inferred).
var ingresses = &model.KindProfile{
	Key:        "ingresses",
	Group:      groupNetworking,
	Version:    "v1",
	Resource:   "ingresses",
	Kind:       "Ingress",
	Title:      "Ingresses",
	Namespaced: true,
	Columns: []model.Column{
		nameColumn,
		namespaceColumn,
		{ID: "loadBalancers", Title: "Load Balancers", Type: model.ColumnText},
		{ID: "rules", Title: "Rules", Type: model.ColumnText},
		ageColumn,
	},
	Cells: ingressCells,
	Fields: map[string]fieldFunc{
		"host": func(obj *unstructured.Unstructured) []string {
			return ingressHosts(obj)
		},
		"path": func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, r := range model.NestedSlice(obj, "spec", "rules") {
				for _, p := range model.NestedSlice(r, "http", "paths") {
					out = appendUnique(out, model.NestedString(p, "path"))
				}
			}
			return out
		},
		"ingressClass": func(obj *unstructured.Unstructured) []string {
			return nonEmpty(ingressClassName(obj))
		},
		"service": func(obj *unstructured.Unstructured) []string {
			return ingressServices(obj)
		},
		"address": func(obj *unstructured.Unstructured) []string {
			return ingressAddresses(obj)
		},
		"tlsSecret": func(obj *unstructured.Unstructured) []string {
			var out []string
			for _, t := range model.NestedSlice(obj, "spec", "tls") {
				out = appendUnique(out, model.NestedString(t, "secretName"))
			}
			return out
		},
	},
	Detail: ingressDetail,
}

func ingressCells(obj *unstructured.Unstructured) ([]model.Cell, model.Status) {
	return []model.Cell{
		nameCell(obj),
		namespaceCell(obj),
		{Text: orDash(loadBalancerIngress(obj))},
		// The cell names several objects (backend services), so it stays text without a link.
		{Text: strings.Join(ingressRules(obj), ", ")},
		model.AgeCell(created(obj)),
	}, model.StatusNone
}

// ingressRules renders each path of each rule as "host/path -> backend" (ASCII: Fyne's bundled
// font has no arrow glyph), a rule without paths as its host, and the default backend last as
// "(default) -> backend". A rule without a host matches every host and reads "*".
func ingressRules(obj any) []string {
	var out []string
	for _, r := range model.NestedSlice(obj, "spec", "rules") {
		host := ingressHost(r)
		paths := model.NestedSlice(r, "http", "paths")
		if len(paths) == 0 {
			out = append(out, host)
			continue
		}
		for _, p := range paths {
			line := host + model.NestedString(p, "path")
			if b := ingressBackend(model.NestedMap(p, "backend")); b != "" {
				line += " -> " + b
			}
			out = append(out, line)
		}
	}
	if b := ingressBackend(model.NestedMap(obj, "spec", "defaultBackend")); b != "" {
		out = append(out, "(default) -> "+b)
	}
	return out
}

func ingressHost(rule any) string {
	if h := model.NestedString(rule, "host"); h != "" {
		return h
	}
	return anyHost
}

// ingressBackend renders a backend as "service:port" (the port's number, else its name), or a
// resource backend as "Kind/name". It returns "" for a missing or empty backend.
func ingressBackend(b map[string]any) string {
	if svc := model.NestedMap(b, "service"); svc != nil {
		name := model.NestedString(svc, "name")
		port := model.NestedIntOrString(svc, "port", "number")
		if port == "" || port == "0" {
			port = model.NestedString(svc, "port", "name")
		}
		if name != "" && port != "" {
			return name + ":" + port
		}
		return name
	}
	if res := model.NestedMap(b, "resource"); res != nil {
		kind, name := model.NestedString(res, "kind"), model.NestedString(res, "name")
		if kind == "" || name == "" {
			return kind + name
		}
		return kind + "/" + name
	}
	return ""
}

// ingressClassName is spec.ingressClassName, else the older class annotation.
func ingressClassName(obj any) string {
	if c := model.NestedString(obj, "spec", "ingressClassName"); c != "" {
		return c
	}
	return model.NestedString(obj, "metadata", "annotations", ingressClassAnnotation)
}

// ingressHosts returns the hosts of the rules, then those of the TLS entries.
func ingressHosts(obj any) []string {
	var out []string
	for _, r := range model.NestedSlice(obj, "spec", "rules") {
		out = appendUnique(out, model.NestedString(r, "host"))
	}
	for _, t := range model.NestedSlice(obj, "spec", "tls") {
		out = appendUnique(out, model.NestedStrings(t, "hosts")...)
	}
	return out
}

// ingressServices returns the names of the backend services, the default backend's included.
func ingressServices(obj any) []string {
	out := appendUnique(nil, model.NestedString(obj, "spec", "defaultBackend", "service", "name"))
	for _, r := range model.NestedSlice(obj, "spec", "rules") {
		for _, p := range model.NestedSlice(r, "http", "paths") {
			out = appendUnique(out, model.NestedString(p, "backend", "service", "name"))
		}
	}
	return out
}

// ingressAddresses returns every IP and hostname of status.loadBalancer.ingress, for agents; the
// Load Balancers cell shows one address per entry, as the Services view does.
func ingressAddresses(obj any) []string {
	var out []string
	for _, in := range model.NestedSlice(obj, "status", "loadBalancer", "ingress") {
		out = appendUnique(out, model.NestedString(in, "ip"), model.NestedString(in, "hostname"))
	}
	return out
}

func ingressDetail(obj *unstructured.Unstructured) []model.DetailSection {
	var d detailItems
	d.add("Class", ingressClassName(obj))
	d.add("Load Balancers", strings.Join(ingressLoadBalancerLines(obj), "\n"))
	d.add("Default Backend", ingressBackendDetail(model.NestedMap(obj, "spec", "defaultBackend")))
	sections := withSection(nil, "Ingress", d)

	var rules detailItems
	for _, r := range model.NestedSlice(obj, "spec", "rules") {
		paths := model.NestedSlice(r, "http", "paths")
		lines := make([]string, 0, len(paths))
		for _, p := range paths {
			line := ingressPathDetail(model.NestedString(p, "path"), model.NestedString(p, "pathType"))
			if b := ingressBackendDetail(model.NestedMap(p, "backend")); b != "" {
				line += " -> " + b
			}
			lines = append(lines, line)
		}
		if len(lines) == 0 {
			lines = append(lines, "no paths: requests go to the default backend")
		}
		rules.add(ingressHost(r), strings.Join(lines, "\n"))
	}
	sections = withSection(sections, "Rules", rules)

	// TLS entries name the Secret that holds the certificate; Cogpit shows the name, never the
	// Secret's data.
	var tls detailItems
	for _, t := range model.NestedSlice(obj, "spec", "tls") {
		secret := model.NestedString(t, "secretName")
		if secret == "" {
			secret = "(no secret)"
		}
		hosts := model.NestedStrings(t, "hosts")
		if len(hosts) == 0 {
			hosts = []string{anyHost}
		}
		tls.add(secret, strings.Join(hosts, "\n"))
	}
	return withSection(sections, "TLS", tls)
}

// ingressPathDetail renders a path with its type: "/api (Prefix)", "(no path, ImplementationSpecific)".
func ingressPathDetail(path, typ string) string {
	switch {
	case path != "" && typ != "":
		return path + " (" + typ + ")"
	case path != "":
		return path
	case typ != "":
		return "(no path, " + typ + ")"
	}
	return "(no path)"
}

// ingressBackendDetail is ingressBackend with a resource backend's API group: "Kind/name
// (group)".
func ingressBackendDetail(b map[string]any) string {
	s := ingressBackend(b)
	if g := model.NestedString(b, "resource", "apiGroup"); g != "" && s != "" {
		s += " (" + g + ")"
	}
	return s
}

// ingressLoadBalancerLines renders each load-balancer entry as its address with the ports the
// load balancer reports, and a port's error: "203.0.113.10 (80/TCP, 443/TCP: error text)".
func ingressLoadBalancerLines(obj any) []string {
	entries := model.NestedSlice(obj, "status", "loadBalancer", "ingress")
	out := make([]string, 0, len(entries))
	for _, in := range entries {
		addr := strings.Join(appendUnique(nil, model.NestedString(in, "ip"), model.NestedString(in, "hostname")), ", ")
		if addr == "" {
			continue
		}
		var ports []string
		for _, p := range model.NestedSlice(in, "ports") {
			port := model.NestedIntOrString(p, "port")
			if port == "" {
				continue
			}
			port += "/" + protocol(p)
			if e := model.NestedString(p, "error"); e != "" {
				port += ": " + singleLine(e)
			}
			ports = append(ports, port)
		}
		if len(ports) > 0 {
			addr += " (" + strings.Join(ports, ", ") + ")"
		}
		out = append(out, addr)
	}
	return out
}

// withSection appends a drawer section when it has items: a field the object does not set leaves
// no empty section behind.
func withSection(sections []model.DetailSection, title string, items detailItems) []model.DetailSection {
	if len(items) == 0 {
		return sections
	}
	return append(sections, model.DetailSection{Title: title, Items: items})
}
