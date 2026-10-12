package cache

import (
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	toolscache "k8s.io/client-go/tools/cache"
)

// The rules a covered scope's view follows, one at a time: what reaches the subscribers for each
// event of its source, and what a handover sends.

func ruleView(t *testing.T) (*view, *[]string) {
	t.Helper()
	scope := Scope{GVR: schema.GroupVersionResource{Version: "v1", Resource: "secrets"}, LabelSelector: "owner=helm"}
	e := &entry{scope: scope, log: slog.Default()}
	var got []string
	e.subs.Store(&[]*subscription{})
	e.subscribe(func(ev Event) {
		got = append(got, fmt.Sprintf("%s %s@%s", ev.Type, ev.Object.GetName(), ev.Object.GetResourceVersion()))
	})
	sel, ok := selectorOf(scope)
	if !ok {
		t.Fatal("no selector")
	}
	e.view = newView(e, sel)
	return e.view, &got
}

func ruleSource(v *view) *source {
	inf := toolscache.NewSharedIndexInformer(&toolscache.ListWatch{}, &unstructured.Unstructured{}, 0, toolscache.Indexers{})
	return &source{v: v, run: &runner{inf: inf}, staging: map[string]*unstructured.Unstructured{}, quit: make(chan struct{})}
}

func obj(name, rv, uid string, helm bool) *unstructured.Unstructured {
	u := &unstructured.Unstructured{Object: map[string]any{}}
	u.SetAPIVersion("v1")
	u.SetKind("Secret")
	u.SetNamespace("a")
	u.SetName(name)
	u.SetResourceVersion(rv)
	u.SetUID(types.UID(uid))
	if helm {
		u.SetLabels(map[string]string{"owner": "helm"})
	} else {
		u.SetLabels(map[string]string{"owner": "someone"})
	}
	return u
}

// apply delivers what apply returns, as the source's handler does.
func applyAll(v *view, typ EventType, u *unstructured.Unstructured, unknown bool) {
	v.mu.Lock()
	evs := v.apply(typ, u, unknown)
	v.mu.Unlock()
	for _, ev := range evs {
		v.e.deliver(ev)
	}
}

func TestViewApplyRules(t *testing.T) {
	v, got := ruleView(t)
	steps := []struct {
		typ     EventType
		u       *unstructured.Unstructured
		unknown bool
		want    string
	}{
		{Added, obj("x", "5", "u1", true), false, "added x@5"},
		{Updated, obj("x", "5", "u1", true), false, ""}, // the same version again
		{Updated, obj("x", "4", "u1", true), false, ""}, // an older version
		{Updated, obj("x", "7", "u1", true), false, "updated x@7"},
		{Updated, obj("y", "8", "u2", false), false, ""},            // never selected
		{Updated, obj("x", "6", "u1", false), false, ""},            // an older version that left the selection
		{Updated, obj("x", "9", "u1", false), false, "deleted x@9"}, // the labels leave: a delete
		{Added, obj("x", "8", "u1", true), false, ""},               // older than the version that removed it
		{Added, obj("x", "10", "u1", true), false, "added x@10"},
		{Deleted, obj("x", "3", "u1", true), false, ""},           // a watch delete older than the version held
		{Deleted, obj("x", "11", "u0", true), false, ""},          // the delete of an older object of that name
		{Deleted, obj("x", "3", "u1", true), true, "deleted x@3"}, // found missing on a relist: final
		{Deleted, obj("x", "12", "u1", true), false, ""},          // not held
	}
	for i, st := range steps {
		before := len(*got)
		applyAll(v, st.typ, st.u, st.unknown)
		var now string
		if len(*got) > before {
			now = strings.Join((*got)[before:], ", ")
		}
		if now != st.want {
			t.Errorf("step %d (%s %s@%s): got %q, want %q", i, st.typ, st.u.GetName(), st.u.GetResourceVersion(), now, st.want)
		}
	}
}

func TestViewHandoverRules(t *testing.T) {
	v, got := ruleView(t)
	for _, u := range []*unstructured.Unstructured{
		obj("gone", "5", "g", true), obj("same", "6", "s", true), obj("newer", "7", "n", true),
		obj("ahead", "20", "a", true), obj("reborn", "8", "r1", true),
	} {
		applyAll(v, Added, u, false)
	}
	applyAll(v, Updated, obj("left", "3", "l", true), false)
	applyAll(v, Added, obj("left", "3", "l", true), false)
	applyAll(v, Updated, obj("left", "15", "l", false), false) // removed at 15
	*got = nil

	src := ruleSource(v)
	for _, u := range []*unstructured.Unstructured{
		obj("same", "6", "s", true), obj("newer", "9", "n", true), obj("ahead", "18", "a", true),
		obj("reborn", "12", "r2", true), obj("left", "3", "l", true), obj("new", "14", "w", true),
	} {
		src.staging[objectKey(u.GetNamespace(), u.GetName())] = u
	}
	src.pos = "21"
	v.mu.Lock()
	evs := v.diff(src, true)
	v.mu.Unlock()
	for _, ev := range evs {
		v.e.deliver(ev)
	}
	want := "deleted gone@5, added new@14, updated newer@9, deleted reborn@8, added reborn@12"
	if s := strings.Join(*got, ", "); s != want {
		t.Errorf("handover:\n got %s\nwant %s", s, want)
	}
	if u, _ := v.get("a/ahead"); u.GetResourceVersion() != "20" {
		t.Errorf("an object the view has newer went back to %s", u.GetResourceVersion())
	}
	if _, ok := v.get("a/left"); ok {
		t.Error("an object removed at 15 came back in its version 3")
	}
	*got = nil
	applyAll(v, Added, obj("gone", "20", "g", true), false) // not newer than the source had come to (21)
	applyAll(v, Added, obj("gone", "22", "g", true), false)
	if s := strings.Join(*got, ", "); s != "added gone@22" {
		t.Errorf("after the handover: %s", s)
	}

	// A handover that timed out after the informer listed keeps what the informer holds but its
	// handler has not delivered; one that timed out before the informer's first list removes
	// nothing (settle does it once the list is there, R-7).
	for _, listed := range []bool{true, false} {
		v2, got2 := ruleView(t)
		applyAll(v2, Added, obj("pending", "5", "p", true), false)
		applyAll(v2, Added, obj("gone", "6", "g", true), false)
		*got2 = nil
		src2 := ruleSource(v2)
		src2.reg = fakeRegistration(listed)
		if err := src2.run.inf.GetStore().Add(obj("pending", "7", "p", true)); err != nil {
			t.Fatal(err)
		}
		v2.mu.Lock()
		evs = v2.diff(src2, false)
		v2.mu.Unlock()
		for _, ev := range evs {
			v2.e.deliver(ev)
		}
		want := "deleted gone@6"
		if !listed {
			want = ""
		}
		if s := strings.Join(*got2, ", "); s != want {
			t.Errorf("timed-out handover (listed %v): %q, want %q", listed, s, want)
		}
		if _, ok := v2.get("a/pending"); !ok {
			t.Errorf("an object the informer holds was dropped (listed %v)", listed)
		}

		// Once the informer has listed, settle removes what it does not hold.
		*got2 = nil
		v2.mu.Lock()
		v2.cur, v2.next = src2, nil
		v2.mu.Unlock()
		src2.reg = fakeRegistration(true)
		v2.settle(src2)
		settled := "deleted gone@6" // what the diff left for the list to settle
		if listed {
			settled = "" // the diff removed it already
		}
		if s := strings.Join(*got2, ", "); s != settled {
			t.Errorf("settle (listed %v): %q, want %q", listed, s, settled)
		}
		if _, ok := v2.get("a/pending"); !ok {
			t.Errorf("settle dropped an object the informer holds (listed %v)", listed)
		}
	}
}

// fakeRegistration is a handler registration that has, or has not, received its informer's
// initial list.
type fakeRegistration bool

func (r fakeRegistration) HasSynced() bool { return bool(r) }

func (r fakeRegistration) HasSyncedChecker() toolscache.DoneChecker { return nil }
