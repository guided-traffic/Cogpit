package model

import (
	"encoding/json"
	"strings"
	"testing"
)

// Agents see a private item as withheld, with nothing that could carry its value; other items
// and sections are untouched, and so is the input.
func TestWithholdPrivate(t *testing.T) {
	in := []DetailSection{
		{Title: TitleProperties, Items: []DetailItem{{Name: "Chart", Value: "web-1.2.0"}}},
		{Title: "Values", Items: []DetailItem{
			{Value: "password: hunter2", Private: true, Chips: []Chip{{Text: "hunter2"}},
				Actions: []ItemAction{{Label: "Copy", Command: "x.copy"}}, Link: &ObjectRef{Kind: "secrets", Name: "x"}},
			{Name: "Note", Value: "user-supplied only"},
		}},
	}
	out := WithholdPrivate(in)
	if &out[0] == &in[0] {
		t.Fatal("the input slice was returned although a section changed")
	}
	if out[0].Items[0].Value != "web-1.2.0" || out[1].Items[1].Value != "user-supplied only" {
		t.Errorf("public items changed: %+v", out)
	}
	got := out[1].Items[0]
	if got.Value != Withheld || got.Chips != nil || got.Actions != nil || got.Link != nil || !got.Private {
		t.Errorf("private item for agents: %+v", got)
	}
	if in[1].Items[0].Value != "password: hunter2" {
		t.Error("the input was modified")
	}
	b, err := json.Marshal(out)
	if err != nil || strings.Contains(string(b), "hunter2") || strings.Contains(string(b), "rivate") {
		t.Errorf("JSON for agents: %s", b)
	}
	plain := []DetailSection{{Title: "A", Items: []DetailItem{{Value: "a"}}}}
	if got := WithholdPrivate(plain); &got[0] != &plain[0] {
		t.Error("sections without a private item were copied")
	}
}
