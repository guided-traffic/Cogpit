package engine

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/guided-traffic/cogpit/internal/metrics"
	"github.com/guided-traffic/cogpit/internal/model"
)

// Hiding metrics (G12, G13; ui-parity › Cluster settings › Hide metrics from the UI): a cluster's
// settings can hide metrics from the window, and what is hidden is not asked of the cluster
// either. The set names where metrics show (the cluster Overview's charts, the nodes', the pods',
// a pod's containers') and what they measure (CPU, memory, the network, the filesystem). A metric
// shows only when neither its place nor its measure is hidden.
//
// Hidden metrics are left out of the lists' CPU and Memory columns (hidden through the snapshot's
// columns, the mechanism ui.setColumns uses), the drawer's chart variants and its containers'
// charts, and the Overview's charts; and a list, a drawer or the Overview whose metrics are all
// hidden holds no subscription of the metrics hub, so no provider is polled for it (cache first:
// nothing is asked for what is not shown). The network and the filesystem are asked of Prometheus
// only while a chart shows them, so hiding them stops those queries. metrics.get still answers an
// agent's explicit read: the setting hides metrics from the UI (Lens's wording), it does not
// forbid reading them.
//
// Which entries the list has is Cogpit's: the places are what Cogpit shows metrics for (Lens's own
// list is not captured), the measures the drawer's variants (the lead's list, 2026-10-13).

// MetricSet is a set of hideable metrics. JSON: the list of their names, in HideableMetrics' order.
type MetricSet uint16

// The hideable metrics.
const (
	// HideCluster hides the cluster Overview's charts (CPU, Memory and Pods).
	HideCluster MetricSet = 1 << iota
	// HideNodes hides the Nodes list's CPU and Memory columns and a node drawer's charts.
	HideNodes
	// HidePods hides the Pods list's CPU and Memory columns and a pod drawer's charts.
	HidePods
	// HideContainers hides the charts of a pod's containers in its drawer.
	HideContainers
	// HideCPU, HideMemory, HideNetwork and HideFilesystem hide one measure everywhere: the CPU
	// and Memory columns and charts, the drawer's Network and Filesystem variants.
	HideCPU
	HideMemory
	HideNetwork
	HideFilesystem

	// HideAll is every hideable metric: "Hide all metrics".
	HideAll = HideCluster | HideNodes | HidePods | HideContainers | HideCPU | HideMemory | HideNetwork | HideFilesystem
)

// HideableMetric is one entry of the Hide metrics setting.
type HideableMetric struct {
	Set   MetricSet
	Name  string // the name commands and the settings file use
	Label string // the page's label
	Place bool   // a place metrics show in, not a measure
}

// HideableMetrics are the entries in the order the page lists them: the places, then the measures.
var HideableMetrics = []HideableMetric{
	{HideCluster, "cluster", "Cluster", true},
	{HideNodes, string(model.UsageNodes), "Nodes", true},
	{HidePods, string(model.UsagePods), "Pods", true},
	{HideContainers, "containers", "Containers", true},
	{HideCPU, "cpu", "CPU", false},
	{HideMemory, "memory", "Memory", false},
	{HideNetwork, "network", "Network", false},
	{HideFilesystem, "filesystem", "Filesystem", false},
}

// hideAllName names every metric at once in ParseMetricSet.
const hideAllName = "all"

// ParseMetricSet reads metric names ("pods", "network", or "all" for every one); an unknown name
// is an error that lists the known ones.
func ParseMetricSet(names []string) (MetricSet, error) {
	var m MetricSet
	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == hideAllName {
			m |= HideAll
			continue
		}
		i := slices.IndexFunc(HideableMetrics, func(h HideableMetric) bool { return h.Name == n })
		if i < 0 {
			return 0, fmt.Errorf("metric %q: one of %s, or all", n, strings.Join(MetricNames(HideAll), ", "))
		}
		m |= HideableMetrics[i].Set
	}
	return m, nil
}

// MetricNames returns the names of a set's metrics, in HideableMetrics' order; never nil.
func MetricNames(m MetricSet) []string {
	out := []string{}
	for _, h := range HideableMetrics {
		if m&h.Set != 0 {
			out = append(out, h.Name)
		}
	}
	return out
}

// Labels returns the page's labels of a set's metrics, in HideableMetrics' order.
func (m MetricSet) Labels() []string {
	out := []string{}
	for _, h := range HideableMetrics {
		if m&h.Set != 0 {
			out = append(out, h.Label)
		}
	}
	return out
}

// Has reports whether every metric of o is in m.
func (m MetricSet) Has(o MetricSet) bool { return m&o == o }

// MarshalJSON writes the set as the list of its names.
func (m MetricSet) MarshalJSON() ([]byte, error) { return json.Marshal(MetricNames(m)) }

// UnmarshalJSON reads a list of names. A name this Cogpit does not know (a file a later one
// wrote) is dropped rather than refused, so that one entry does not cost the others; commands
// check their names with ParseMetricSet.
func (m *MetricSet) UnmarshalJSON(b []byte) error {
	var names []string
	if err := json.Unmarshal(b, &names); err != nil {
		return err
	}
	*m = 0
	for _, n := range names {
		if s, err := ParseMetricSet([]string{n}); err == nil {
			*m |= s
		}
	}
	return nil
}

// usagePlace is the place of a usage kind's metrics: the Pods' or the Nodes'.
func usagePlace(k model.UsageKind) MetricSet {
	if k == model.UsageNodes {
		return HideNodes
	}
	return HidePods
}

// listColumns reports which of a usage kind's CPU and Memory columns the set leaves visible.
func (m MetricSet) listColumns(k model.UsageKind) (cpu, memory bool) {
	if k == "" || m&usagePlace(k) != 0 {
		return false, false
	}
	return m&HideCPU == 0, m&HideMemory == 0
}

// drawerVariants are the chart variants a usage kind's drawer shows, in the buttons' order; none
// when its place is hidden.
func (m MetricSet) drawerVariants(k model.UsageKind) []string {
	if k == "" || m&usagePlace(k) != 0 {
		return nil
	}
	out := make([]string, 0, len(MetricsVariants))
	for _, v := range MetricsVariants {
		if m&variantSet(v) == 0 {
			out = append(out, v)
		}
	}
	return out
}

// variantSet is the measure of a chart variant.
func variantSet(variant string) MetricSet {
	switch variant {
	case VariantMemory:
		return HideMemory
	case VariantNetwork:
		return HideNetwork
	case VariantFilesystem:
		return HideFilesystem
	}
	return HideCPU
}

// overviewShown reports which of the Overview's charts the set leaves visible: CPU, Memory, Pods.
func (m MetricSet) overviewShown() (cpu, memory, pods bool) {
	if m&HideCluster != 0 {
		return false, false, false
	}
	return m&HideCPU == 0, m&HideMemory == 0, true
}

// StatusHidden is the metrics state of a section whose metrics the cluster's settings hide.
const StatusHidden metrics.Status = "hidden"

// msgHidden says why a section shows no metrics.
const msgHidden = "hidden in this cluster's settings (Cluster Settings › Metrics)"

// hiddenMetricsOf returns the metrics a context's settings hide, as the UI state holds them.
func hiddenMetricsOf(u *UIState, context string) MetricSet {
	if u == nil || context == "" {
		return 0
	}
	return u.ClusterSettings[context].HiddenMetrics
}

// hiddenMetrics returns the metrics a context's settings hide, from the UI state.
func (s *Session) hiddenMetrics(context string) MetricSet {
	s.mu.Lock()
	defer s.mu.Unlock()
	return hiddenMetricsOf(&s.ui, context)
}

// hideMetricColumns marks a usage kind's CPU and Memory columns hidden when the set hides them,
// and returns the ids of those the view itself showed (TableSnapshot.MetricsHidden). The columns
// keep their indexes, so the rows still fit them. cols is the snapshot's own copy.
func hideMetricColumns(p *model.KindProfile, cols []model.Column, hidden MetricSet) ([]model.Column, []string) {
	if p.Usage == "" || hidden == 0 {
		return cols, nil
	}
	cpu, memory := hidden.listColumns(p.Usage)
	var ids []string
	for i := range cols {
		switch cols[i].ID {
		case model.ColumnCPU:
			if !cpu && !cols[i].Hidden {
				cols = cloneOnce(cols, p.Columns)
				cols[i].Hidden = true
				ids = append(ids, cols[i].ID)
			}
		case model.ColumnMemory:
			if !memory && !cols[i].Hidden {
				cols = cloneOnce(cols, p.Columns)
				cols[i].Hidden = true
				ids = append(ids, cols[i].ID)
			}
		}
	}
	return cols, ids
}

// cloneOnce returns cols, cloned first when it is the profile's own slice (ColumnsShown returns
// it as it is when the view hides nothing of its own).
func cloneOnce(cols, profile []model.Column) []model.Column {
	if len(cols) > 0 && len(profile) > 0 && &cols[0] == &profile[0] {
		return slices.Clone(cols)
	}
	return cols
}

// listFeedWanted reports whether a view of p polls its metrics with this set hidden: only while
// one of its CPU and Memory columns shows.
func listFeedWanted(p *model.KindProfile, hidden MetricSet) bool {
	cpu, memory := hidden.listColumns(p.Usage)
	return cpu || memory
}

// followHidden starts or stops the view's metrics feed as the hidden metrics say: a list whose CPU
// and Memory columns are both hidden polls nothing.
func (w *viewWorker) followHidden(hidden MetricSet) {
	if w.profile.Usage == "" || w.stopped.Load() {
		return
	}
	want := listFeedWanted(w.profile, hidden)
	w.mu.Lock()
	cur := w.usage
	switch {
	case want && cur == nil:
		w.usage = w.backend.startUsageFeed(w.profile, func() { signal(w.wake) })
	case !want && cur != nil:
		w.usage = nil
	}
	w.mu.Unlock()
	if !want && cur != nil {
		cur.release()
	}
}

// followHidden starts or stops the drawer's metrics poll as the hidden metrics say, and tells the
// poll which series its visible charts show. A drawer whose charts are all hidden polls nothing.
// When what the drawer shows changes, the window is told (its section and its variants).
func (d *drawerWatch) followHidden(s *Session, hidden MetricSet) {
	if d.profile.Usage == "" || d.stopped.Load() {
		return
	}
	want := len(hidden.drawerVariants(d.profile.Usage)) > 0
	d.wmu.Lock()
	changed := d.hidden != hidden || !d.hiddenSet
	d.hidden, d.hiddenSet = hidden, true
	cur := d.usage
	d.wmu.Unlock()
	switch {
	case !want && cur != nil:
		d.wmu.Lock()
		d.usage = nil
		d.wmu.Unlock()
		cur.Release()
	case want && cur == nil:
		sub := s.startDrawerUsage(d.backend, d.profile, d.ref)
		d.wmu.Lock()
		d.usage = sub
		windows, wants := d.windowListLocked(), d.wantsLocked()
		d.wmu.Unlock()
		if len(windows) > 0 {
			sub.SetWindows(windows...)
		}
		sub.SetWants(wants)
	case cur != nil && changed:
		d.wmu.Lock()
		wants := d.wantsLocked()
		d.wmu.Unlock()
		cur.SetWants(wants)
	}
	if changed {
		// Off the caller's locks (the views' state): subscribers may read the engine's state.
		kind := d.ref.Kind
		s.goSafe("hidden metrics", func() {
			s.emit(Event{Type: EventDetail, View: kind})
			s.emit(Event{Type: EventMetrics, View: kind})
		})
	}
}

// windowListLocked is the charts' time ranges set; d.wmu is held.
func (d *drawerWatch) windowListLocked() []time.Duration {
	list := make([]time.Duration, 0, len(d.windows))
	for _, w := range d.windows {
		list = append(list, w)
	}
	return list
}

// wantsLocked is what the drawer's visible charts show beyond CPU and memory: the network and the
// filesystem of a chart that shows them, unless they are hidden. d.wmu is held.
func (d *drawerWatch) wantsLocked() metrics.Want {
	shows := slices.Collect(maps.Values(d.variants))
	if visible := d.hidden.drawerVariants(d.profile.Usage); len(visible) > 0 && d.hidden.Has(HideCPU|HideMemory) {
		shows = append(shows, visible[0]) // a chart left at its default shows the first visible variant
	}
	var want metrics.Want
	for _, v := range shows {
		if d.hidden&variantSet(v) != 0 {
			continue
		}
		switch v {
		case VariantNetwork:
			want |= metrics.WantNetwork
		case VariantFilesystem:
			want |= metrics.WantFilesystem
		}
	}
	return want
}

// defaultVariant is the variant of a chart nobody set: CPU, else the first visible one the
// providers have, else the first visible one.
func defaultVariant(visible []string, caps metrics.Capabilities) string {
	if shown := shownVariants(visible, caps); len(shown) > 0 {
		return shown[0]
	}
	if len(visible) > 0 {
		return visible[0]
	}
	return VariantCPU
}

// usageSub returns the drawer's metrics poll; nil while none runs.
func (d *drawerWatch) usageSub() *metrics.Subscription {
	d.wmu.Lock()
	defer d.wmu.Unlock()
	return d.usage
}

// shownVariants returns the variants the drawer's charts may show: the visible ones the
// providers have. A chart whose variant is hidden, or one the providers lack, shows the first of
// these.
func shownVariants(visible []string, caps metrics.Capabilities) []string {
	out := make([]string, 0, len(visible))
	for _, v := range visible {
		switch {
		case v == VariantNetwork && !caps.Network, v == VariantFilesystem && !caps.Filesystem:
			continue
		}
		out = append(out, v)
	}
	return out
}

// markHiddenCharts empties and marks the Overview's charts the set hides (CPU, Memory, Pods, in
// the model's order) and reports whether all of them are hidden; then the model's state says so.
func markHiddenCharts(m *ClusterOverviewModel, hidden MetricSet) (all bool) {
	cpu, memory, pods := hidden.overviewShown()
	shown := []bool{cpu, memory, pods}
	for i := range m.Charts {
		if i < len(shown) && !shown[i] {
			c := m.Charts[i]
			m.Charts[i] = OverviewChart{Metric: c.Metric, Title: c.Title, Unit: c.Unit, Hidden: true}
		}
	}
	if !cpu && !memory && !pods {
		m.Status, m.Message = StatusHidden, msgHidden
		return true
	}
	return false
}

// setHidden follows a change of the hidden metrics while the Overview is open: the cluster scope
// is released when its CPU and Memory charts are both hidden, and subscribed again when one shows.
func (c *overviewCharts) setHidden(h MetricSet) {
	c.mu.Lock()
	if c.closed || c.hidden == h {
		c.mu.Unlock()
		return
	}
	c.hidden = h
	var old *metrics.Subscription
	cpu, memory, _ := h.overviewShown()
	switch {
	case !cpu && !memory && c.sub != nil:
		old, c.sub, c.scope = c.sub, nil, metrics.Scope{}
	case (cpu || memory) && c.sub == nil && c.last != nil && c.last.nodesSynced && c.b.Metrics != nil:
		c.subscribeLocked(c.last)
	}
	c.mu.Unlock()
	old.Release()
	c.s.goSafe("hidden metrics", func() { c.s.emit(Event{Type: EventMetrics, View: ClusterOverview}) })
}
