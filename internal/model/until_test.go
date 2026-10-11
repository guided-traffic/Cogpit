package model

import (
	"testing"
	"time"
)

func TestFormatUntil(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{-time.Minute, "now"},
		{0, "now"},
		{30 * time.Second, "In less than a minute"},
		{12 * time.Minute, "In about 12 min"},
		{time.Hour, "In about 1 hour"},
		{time.Hour + 57*time.Minute + 20*time.Second, "In about 1 hour 57 min"}, // as captured from Lens
		{5 * time.Hour, "In about 5 hours"},
		{24 * time.Hour, "In about 1 day"},
		{51 * time.Hour, "In about 2 days 3 hours"},
	}
	for _, tt := range tests {
		if got := FormatUntil(tt.d); got != tt.want {
			t.Errorf("FormatUntil(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
	col := Column{ID: "nextExecution", Type: ColumnUntil}
	if got := CellText(col, Cell{Num: now.Add(2 * time.Hour).Unix()}, now); got != "In about 2 hours" {
		t.Errorf("CellText of an until cell = %q", got)
	}
	if got := CellText(col, Cell{Text: "-"}, now); got != "-" {
		t.Errorf("CellText of an empty until cell = %q", got)
	}
}
