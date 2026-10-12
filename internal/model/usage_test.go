package model_test

import (
	"testing"

	"github.com/guided-traffic/cogpit/internal/model"
)

func TestUsageFormats(t *testing.T) {
	for _, tc := range []struct{ got, want string }{
		{model.FormatCPU(0), "0.000"},
		{model.FormatCPU(12), "0.012"},
		{model.FormatCPU(1500), "1.500"},
		{model.FormatBytes(0), "0B"},
		{model.FormatBytes(1023), "1023B"},
		{model.FormatBytes(1024), "1.0KiB"},
		{model.FormatBytes(20_447_232), "19.5MiB"},
		{model.FormatBytes(3 << 30), "3.0GiB"},
		{model.FormatPercent(0.075), "7.5%"},
		{model.FormatPercent(0.424), "42%"},
		{model.FormatPercent(1.2), "120%"},
	} {
		if tc.got != tc.want {
			t.Errorf("got %q, want %q", tc.got, tc.want)
		}
	}
}

func TestPodUsageTotal(t *testing.T) {
	p := model.PodUsage{Containers: []model.ContainerUsage{
		{Name: "a", Usage: model.Usage{CPUMilli: 10, MemoryBytes: 100}},
		{Name: "b", Usage: model.Usage{CPUMilli: 5, MemoryBytes: 1}},
	}}
	if got := p.Total(); got != (model.Usage{CPUMilli: 15, MemoryBytes: 101}) {
		t.Errorf("total = %+v", got)
	}
}
