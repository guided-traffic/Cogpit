package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The Prometheus a cluster's metrics come from, as the user sets it (G12; Lens: Cluster settings →
// Metrics, "Prometheus service address" and a path prefix): a service namespace/service:port and
// a route prefix that replace detection by name, or nothing, and detection finds one. It is kept
// per context in Cogpit's own preferences, <user config dir>/Cogpit/metrics.json, owner-only:
// never in the kubeconfig and never in the cluster (D-2: no implicit kubeconfig writes). Setting
// it sends nothing; the next poll that needs Prometheus probes that service alone (one buildinfo
// GET through the service proxy).

// MetricsSettingsFile is the preferences file of the Prometheus overrides.
const MetricsSettingsFile = "metrics.json"

// maxMetricsSettingsSize bounds the file Cogpit reads back.
const maxMetricsSettingsSize = 1 << 20

// MetricsSettings keeps each context's Prometheus override. Safe for concurrent use. A nil
// MetricsSettings keeps nothing: an override then lasts for the connection.
type MetricsSettings struct {
	mu   sync.Mutex
	path string
	by   map[string]metrics.Override
}

type metricsSettingsFile struct {
	Prometheus map[string]metrics.Override `json:"prometheus"`
}

// NewMetricsSettings reads the overrides kept in dir (MetricsSettingsFile). A file that cannot be
// read is left alone and starts empty; err says why.
func NewMetricsSettings(dir string) (*MetricsSettings, error) {
	m := &MetricsSettings{path: filepath.Join(dir, MetricsSettingsFile), by: map[string]metrics.Override{}}
	var f metricsSettingsFile
	if err := readPrivateJSON(m.path, maxMetricsSettingsSize, &f); err != nil {
		return m, err
	}
	for ctx, o := range f.Prometheus {
		if CheckPrometheusOverride(o) == nil {
			m.by[ctx] = o
		}
	}
	return m, nil
}

// Override returns the override kept for a context, or nil.
func (m *MetricsSettings) Override(context string) *metrics.Override {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	o, ok := m.by[context]
	if !ok {
		return nil
	}
	return &o
}

// SetOverride keeps (or, with nil, forgets) a context's override and writes the file.
func (m *MetricsSettings) SetOverride(context string, o *metrics.Override) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	next := maps.Clone(m.by)
	if o == nil {
		delete(next, context)
	} else {
		next[context] = *o
	}
	data, err := json.MarshalIndent(metricsSettingsFile{Prometheus: next}, "", "  ")
	if err != nil {
		return err
	}
	if err := writePrivateFile(m.path, data); err != nil {
		return fmt.Errorf("keep the Prometheus setting: %w", err)
	}
	m.by = next
	return nil
}

// CheckPrometheusOverride refuses what is not a service address: the namespace and the service
// are DNS labels, the port a number (1 to 65535) or a port name, the prefix a path of letters,
// digits, "-", "_", "." and "/" without "." or ".." segments, so that it can only name a path on
// that service behind the service proxy.
func CheckPrometheusOverride(o metrics.Override) error {
	if errs := validation.IsDNS1123Label(o.Namespace); len(errs) > 0 {
		return fmt.Errorf("namespace %q: %s", o.Namespace, strings.Join(errs, "; "))
	}
	if errs := validation.IsDNS1123Label(o.Service); len(errs) > 0 {
		return fmt.Errorf("service %q: %s", o.Service, strings.Join(errs, "; "))
	}
	if n, err := strconv.Atoi(o.Port); err == nil {
		if n < 1 || n > 65535 {
			return fmt.Errorf("port %d: a port is 1 to 65535", n)
		}
	} else if errs := validation.IsValidPortName(o.Port); len(errs) > 0 {
		return fmt.Errorf("port %q: a number or a port name: %s", o.Port, strings.Join(errs, "; "))
	}
	if len(o.Prefix) > 200 {
		return errors.New("the path prefix is longer than 200 characters")
	}
	if strings.IndexFunc(o.Prefix, notPathRune) >= 0 {
		return fmt.Errorf("path prefix %q: only letters, digits, -, _, . and /", o.Prefix)
	}
	for _, seg := range strings.Split(o.Prefix, "/") {
		if seg == "." || seg == ".." {
			return fmt.Errorf("path prefix %q: no . or .. segments", o.Prefix)
		}
	}
	return nil
}

// notPathRune reports a rune a path prefix may not have.
func notPathRune(r rune) bool {
	letter := r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
	return !letter && !strings.ContainsRune("-_./", r)
}

// ParsePrometheusAddress reads Lens's form of the address, "namespace/service:port".
func ParsePrometheusAddress(s string) (namespace, service, port string, err error) {
	s = strings.TrimSpace(s)
	ns, rest, ok := strings.Cut(s, "/")
	if !ok {
		return "", "", "", fmt.Errorf("%q: write the address as namespace/service:port", s)
	}
	svc, port, ok := strings.Cut(rest, ":")
	if !ok || port == "" {
		return "", "", "", fmt.Errorf("%q: the address needs a port: namespace/service:port", s)
	}
	return ns, svc, port, nil
}

// PrometheusSetting is the Prometheus a connected cluster's metrics come from: the override, or
// detection by name (Auto).
type PrometheusSetting struct {
	// Context is the connected context the setting belongs to.
	Context string `json:"context"`
	// Auto: no override; Prometheus is detected among the cluster's Services by name.
	Auto bool `json:"auto"`
	// Address is the override as Lens writes it, "namespace/service:port"; Prefix its route prefix.
	Address string `json:"address,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	// Kept says whether the setting is in Cogpit's preferences (it outlives the connection); false
	// when Cogpit has no configuration directory.
	Kept bool `json:"kept"`
}

func settingOf(context string, o *metrics.Override, kept bool) PrometheusSetting {
	s := PrometheusSetting{Context: context, Auto: o == nil, Kept: kept}
	if o != nil {
		s.Address, s.Prefix = o.Namespace+"/"+o.Service+":"+o.Port, o.Prefix
	}
	return s
}

// PrometheusSetting returns the connected cluster's Prometheus setting.
func (s *Session) PrometheusSetting() (PrometheusSetting, error) {
	b := s.Backend()
	if b == nil {
		return PrometheusSetting{}, ErrNotConnected
	}
	if b.PrometheusDetector == nil {
		return PrometheusSetting{}, fmt.Errorf("%w: %s has no Prometheus detection (the demo cluster's metrics are its own)", ErrNotApplicable, b.Context)
	}
	return settingOf(b.Context, b.PrometheusDetector.Override(), b.MetricsSettings != nil), nil
}

// SetPrometheusOverride sets the connected cluster's Prometheus (nil: detect by name again),
// keeps it in Cogpit's preferences for that context, and makes the next poll that needs
// Prometheus detect again: a drawer that is open asks the new one for its history at once. It
// writes nothing to the cluster or the kubeconfig and sends no request itself.
func (s *Session) SetPrometheusOverride(o *metrics.Override) (PrometheusSetting, error) {
	b := s.Backend()
	if b == nil {
		return PrometheusSetting{}, ErrNotConnected
	}
	if b.PrometheusDetector == nil || b.Metrics == nil {
		return PrometheusSetting{}, fmt.Errorf("%w: %s has no Prometheus detection (the demo cluster's metrics are its own)", ErrNotApplicable, b.Context)
	}
	if o != nil {
		c := *o
		c.Prefix = strings.Trim(c.Prefix, "/")
		if c.Prefix != "" {
			c.Prefix = "/" + c.Prefix
		}
		if err := CheckPrometheusOverride(c); err != nil {
			return PrometheusSetting{}, err
		}
		o = &c
	}
	if err := b.MetricsSettings.SetOverride(b.Context, o); err != nil {
		return PrometheusSetting{}, err
	}
	b.PrometheusDetector.SetOverride(o)
	b.Metrics.Forget()
	s.emit(Event{Type: EventMetrics})
	return settingOf(b.Context, b.PrometheusDetector.Override(), b.MetricsSettings != nil), nil
}
