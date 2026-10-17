package engine

import (
	"encoding/json"
	"fmt"
	"maps"
	"strings"

	"github.com/guided-traffic/cogpit/internal/alerting"
	"github.com/guided-traffic/cogpit/internal/metrics"
)

// The Alertmanager a cluster's alerts come from, as the user sets it (D-77): a service
// namespace/service:port, a route prefix and HTTPS, which replace detection by name, or nothing,
// and detection finds one. It is kept per context beside the Prometheus address, in Cogpit's own
// metrics.json (metricssettings.go), owner-only: never in the kubeconfig and never in the cluster
// (D-2). Setting it sends nothing itself; the next poll, at once while the cluster is shown, asks
// that service alone.

// Alertmanager returns the address kept for a context, or nil (detection).
func (m *MetricsSettings) Alertmanager(context string) *alerting.Address {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.am[context]
	if !ok {
		return nil
	}
	return &a
}

// SetAlertmanager keeps (or, with nil, forgets) a context's Alertmanager address and writes the
// file; the Prometheus settings stay.
func (m *MetricsSettings) SetAlertmanager(context string, a *alerting.Address) error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	next := maps.Clone(m.am)
	if next == nil {
		next = map[string]alerting.Address{}
	}
	if a == nil {
		delete(next, context)
	} else {
		next[context] = *a
	}
	data, err := json.MarshalIndent(metricsSettingsFile{Prometheus: m.by, Query: m.query, Alertmanager: next}, "", "  ")
	if err != nil {
		return err
	}
	if err := writePrivateFile(m.path, data); err != nil {
		return fmt.Errorf("keep the Alertmanager setting: %w", err)
	}
	m.am = next
	return nil
}

// CheckAlertmanagerAddress refuses what is not a service address, by the rules of the Prometheus
// address (CheckPrometheusOverride): DNS labels, a port number or name, a path prefix of letters,
// digits, "-", "_", "." and "/" without "." or ".." segments.
func CheckAlertmanagerAddress(a alerting.Address) error {
	return CheckPrometheusOverride(metrics.Override{Namespace: a.Namespace, Service: a.Service, Port: a.Port, Prefix: a.Prefix})
}

// AlertmanagerSetting is the Alertmanager a connected cluster's alerts come from: the address the
// user set, or detection by name (Auto), and what Cogpit knows of it now.
type AlertmanagerSetting struct {
	// Context is the connected context the setting belongs to.
	Context string `json:"context"`
	// Auto: no address; Alertmanager is detected among the cluster's Services by name.
	Auto bool `json:"auto"`
	// Address is the address as "namespace/service:port"; Prefix its route prefix; HTTPS asks the
	// service over TLS through the service proxy.
	Address string `json:"address,omitempty"`
	Prefix  string `json:"prefix,omitempty"`
	HTTPS   bool   `json:"https,omitempty"`
	// Kept says whether the setting is in Cogpit's preferences (it outlives the connection); false
	// when Cogpit has no configuration directory.
	Kept bool `json:"kept"`
	// State is what Cogpit knows of the Alertmanager now, InUse the one it reads
	// ("monitoring/kube-prometheus-alertmanager:http-web"), Detail why it reads none. Read from the
	// last poll: no request.
	State  AlertsState `json:"state"`
	InUse  string      `json:"inUse,omitempty"`
	Detail string      `json:"detail,omitempty"`
}

// AlertmanagerSetting returns a connected cluster's Alertmanager setting ("" the shown cluster).
func (s *Session) AlertmanagerSetting(contextID string) (AlertmanagerSetting, error) {
	b, err := s.TargetBackend(contextID)
	if err != nil {
		return AlertmanagerSetting{}, err
	}
	if why := noAlertmanager(b); why != "" {
		return AlertmanagerSetting{}, fmt.Errorf("%w: %s", ErrNotApplicable, why)
	}
	return s.alertmanagerSettingOf(b), nil
}

func (s *Session) alertmanagerSettingOf(b *Backend) AlertmanagerSetting {
	out := AlertmanagerSetting{Context: b.Context, Auto: true, Kept: b.MetricsSettings != nil, State: AlertsChecking}
	a := b.MetricsSettings.Alertmanager(b.Context)
	if p := b.alerts.current(); p != nil {
		a = p.Address() // the one in use: without a settings file it lasts for the connection
	}
	if a != nil {
		out.Auto, out.Address, out.Prefix, out.HTTPS = false, a.Namespace+"/"+a.Service+":"+a.Port, a.Prefix, a.HTTPS
	}
	if u := s.UIState(); !u.IntegrationOn(b.Context, IntegrationAlertmanager) {
		out.State, out.Detail = "", "the Alertmanager integration is off for this cluster"
		return out
	}
	if e, ok := s.alerts.entry(b.Context); ok {
		sum := e.summary
		out.State, out.InUse, out.Detail = sum.State, sum.Where, sum.Detail
	}
	return out
}

// SetAlertmanagerSetting sets a connected cluster's Alertmanager ("" the shown one; nil: detect by
// name again), keeps it in Cogpit's preferences for that context, and makes the cluster's poller
// ask it at once while the cluster is shown. It writes nothing to the cluster or the kubeconfig and
// sends no request itself.
func (s *Session) SetAlertmanagerSetting(contextID string, a *alerting.Address) (AlertmanagerSetting, error) {
	b, err := s.TargetBackend(contextID)
	if err != nil {
		return AlertmanagerSetting{}, err
	}
	if why := noAlertmanager(b); why != "" {
		return AlertmanagerSetting{}, fmt.Errorf("%w: %s", ErrNotApplicable, why)
	}
	if a != nil {
		c := *a
		c.Prefix = strings.Trim(c.Prefix, "/")
		if c.Prefix != "" {
			c.Prefix = "/" + c.Prefix
		}
		if err := CheckAlertmanagerAddress(c); err != nil {
			return AlertmanagerSetting{}, err
		}
		a = &c
	}
	if err := b.MetricsSettings.SetAlertmanager(b.Context, a); err != nil {
		return AlertmanagerSetting{}, err
	}
	if p := b.alerts.current(); p != nil {
		p.SetAddress(a)
	}
	return s.alertmanagerSettingOf(b), nil
}
