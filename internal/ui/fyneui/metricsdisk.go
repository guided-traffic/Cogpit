package fyneui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"

	"github.com/guided-traffic/cogpit/internal/engine"
	"github.com/guided-traffic/cogpit/internal/model"
)

// A node's disk in its drawer (G12): under the node's chart, "Disk (/): 41.2 GiB of 58.4 GiB used
// (71%)" in the usage bars' colour at the thresholds (the warning colour from 80%, the error colour
// from 90%; else the foreground), or quietly why there is none ("Disk: … no Prometheus was
// found"). The numbers are the Nodes list's Disk column's: node-exporter through Prometheus.

func newDiskLine() *textLine {
	l := newTextLine("", theme.ColorNameForeground, theme.SizeNameCaptionText, fyne.TextStyle{})
	l.truncate = true
	l.Hide()
	return l
}

// diskText is a node's disk as the drawer writes it.
func diskText(d model.NodeDisk) string {
	return "Disk (/): " + model.FormatBytes(d.UsedBytes) + " of " + model.FormatBytes(d.SizeBytes) + " used (" + model.FormatPercent(d.Share()) + ")"
}

// showDisk shows the node's disk, or why there is none; nothing for a pod or a hidden disk.
func (p *metricsPanel) showDisk(m *engine.ObjectMetrics) {
	switch {
	case m == nil || m.Disk == nil && m.DiskNote == "":
		p.disk.Hide()
		return
	case m.Disk != nil:
		tok := theme.ColorNameForeground
		if s := model.UsageBarStatus(m.Disk.Share()); s != model.StatusNone {
			tok = statusColorName(s)
		}
		p.disk.SetText(diskText(*m.Disk))
		p.disk.SetColor(tok)
	default:
		p.disk.SetText("Disk: " + m.DiskNote)
		p.disk.SetColor(colorMuted)
	}
	p.disk.Show()
}
