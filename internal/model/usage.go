package model

import (
	"fmt"
	"strconv"
	"time"
)

// Usage is what a pod, a container or a node uses right now, as the Metrics API reports it: CPU
// in millicores and memory in bytes. The zero value is "nothing measured".
type Usage struct {
	CPUMilli    int64 `json:"cpuMilli"`
	MemoryBytes int64 `json:"memoryBytes"`
}

// Add returns the sum of two usages.
func (u Usage) Add(o Usage) Usage {
	return Usage{CPUMilli: u.CPUMilli + o.CPUMilli, MemoryBytes: u.MemoryBytes + o.MemoryBytes}
}

// ContainerUsage is one container's usage.
type ContainerUsage struct {
	Name string `json:"name"`
	Usage
}

// PodUsage is one pod's measurement: its containers' usage at Timestamp, the time the kubelet
// took it (the Metrics API's timestamp, not the time Cogpit asked).
type PodUsage struct {
	Namespace  string           `json:"namespace"`
	Name       string           `json:"name"`
	Timestamp  time.Time        `json:"timestamp"`
	Containers []ContainerUsage `json:"containers"`
}

// Total is the pod's usage: the sum of its containers'.
func (p PodUsage) Total() Usage {
	var u Usage
	for _, c := range p.Containers {
		u = u.Add(c.Usage)
	}
	return u
}

// NodeUsage is one node's measurement at Timestamp.
type NodeUsage struct {
	Name      string    `json:"name"`
	Timestamp time.Time `json:"timestamp"`
	Usage
}

// UsageSample is one point of a usage time series.
type UsageSample struct {
	At time.Time `json:"at"`
	Usage
}

// FormatCPU formats millicores as cores the way Lens's lists do, three decimals: 12 → "0.012",
// 1500 → "1.500".
func FormatCPU(milli int64) string {
	return strconv.FormatFloat(float64(milli)/1000, 'f', 3, 64)
}

// FormatBytes formats a byte count in binary units with one decimal: "19.5MiB", "1.2GiB"; below
// one KiB in bytes ("512B").
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit && exp < 4; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTP"[exp])
}

// FormatPercent formats a share of a whole with one decimal below 10 percent and none above:
// 0.075 → "7.5%", 0.42 → "42%".
func FormatPercent(share float64) string {
	p := share * 100
	if p < 10 {
		return strconv.FormatFloat(p, 'f', 1, 64) + "%"
	}
	return strconv.FormatFloat(p, 'f', 0, 64) + "%"
}
