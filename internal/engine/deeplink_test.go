package engine

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/guided-traffic/cogpit/internal/model"
)

// The grammar of a cogpit:// link (D-47): what parses, and into what.
func TestParseDeepLink(t *testing.T) {
	for _, tc := range []struct {
		link string
		want DeepLink
	}{
		{"cogpit://open?context=kind-dev", DeepLink{Context: "kind-dev"}},
		{"cogpit://open/?view=pods", DeepLink{View: "pods"}},
		{"COGPIT://Open?view=cluster/overview", DeepLink{View: "cluster/overview"}},
		{"cogpit://open?context=kind-dev+%5Bconfig%5D&view=pods&namespace=team-a&name=web-0",
			DeepLink{Context: "kind-dev [config]", View: "pods", Namespace: "team-a", Name: "web-0"}},
		{"cogpit://open?context=arn%3Aaws%3Aeks%3Aeu-central-1%3A1%3Acluster%2Fprod&view=gateways.gateway.networking.k8s.io&name=main",
			DeepLink{Context: "arn:aws:eks:eu-central-1:1:cluster/prod", View: "gateways.gateway.networking.k8s.io", Name: "main"}},
		{"cogpit://open?context=kind-dev%20%5Bconfig%5D&view=pods&namespace=team-a&name=web-0", // usage.md's example
			DeepLink{Context: "kind-dev [config]", View: "pods", Namespace: "team-a", Name: "web-0"}},
		{"cogpit://open?view=clusterroles&name=system:aggregate-to-admin", DeepLink{View: "clusterroles", Name: "system:aggregate-to-admin"}},
		{"cogpit://open?namespace=kube-system", DeepLink{Namespace: "kube-system"}},
		{"cogpit://open?context=%C3%BCber", DeepLink{Context: "über"}},
	} {
		got, err := ParseDeepLink(tc.link)
		if err != nil || got != tc.want {
			t.Errorf("%s: %+v, %v; want %+v", tc.link, got, err, tc.want)
		}
	}
}

// What a link must not be: anything but cogpit://open with the four parameters, each once, not
// empty, in its form and bounds; a name without a kind's view; nothing to open.
func TestParseDeepLinkRefusals(t *testing.T) {
	long := "cogpit://open?view=pods&name=" + strings.Repeat("a", MaxDeepLink)
	for _, tc := range []struct{ link, says string }{
		{"", "scheme"},
		{"open?view=pods", "scheme"},
		{"https://open?view=pods", "scheme"},
		{"file:///etc/passwd", "scheme"},
		{"cogpit:open?view=pods", "write it as"},
		{"cogpit://user:pw@open?view=pods", "user information"},
		{"cogpit://open:80?view=pods", "action"},
		{"cogpit://delete?view=pods&name=web", "action"},
		{"cogpit://connect?context=dev", "action"},
		{"cogpit://open/pods?name=web", "path"},
		{"cogpit://open?view=pods#name=web", "fragment"},
		{"cogpit://open?view=pods%23x", "view"},
		{"cogpit://open", "nothing to open"},
		{"cogpit://open?", "nothing to open"},
		{"cogpit://open?view=pods&view=nodes", "2 times"},
		{"cogpit://open?view=", "empty"},
		{"cogpit://open?view=pods&command=delete", `"command"`},
		{"cogpit://open?view=pods&file=%2Fetc%2Fpasswd", `"file"`},
		{"cogpit://open?view=pods;name=web", "do not parse"},
		{"cogpit://open?view=pods&name=a%zz", "do not parse"},
		{"cogpit://open?view=Pods", "view"},
		{"cogpit://open?view=../pods", "view"},
		{"cogpit://open?view=a/b/c", "view"},
		{"cogpit://open?view=pods/", "view"},
		{"cogpit://open?namespace=Team_A", "namespace"},
		{"cogpit://open?namespace=" + strings.Repeat("a", 64), "namespace"},
		{"cogpit://open?view=pods&name=a/b", "name"},
		{"cogpit://open?view=pods&name=..", "name"},
		{"cogpit://open?view=pods&name=" + strings.Repeat("a", 254), "name"},
		{"cogpit://open?view=pods&name=web%20one", "name"},
		{"cogpit://open?name=web", "needs the view"},
		{"cogpit://open?view=workloads/overview&name=web", "page"},
		{"cogpit://open?context=dev%0Aevil", "control characters"},
		{"cogpit://open?context=dev%E2%80%AEved", "control characters"},
		{"cogpit://open?context=%FF", "invalid UTF-8"},
		{"cogpit://open?context=" + strings.Repeat("c", maxLinkContext+1), "bytes long"},
		{"cogpit://open?view=pods\n", "spaces"},
		{"cogpit://open?view=pods name=x", "spaces"},
		{"cogpit://open?view=pods\x00", "spaces"},
		{long, "at most 2048"},
	} {
		got, err := ParseDeepLink(tc.link)
		if !errors.Is(err, ErrDeepLinkInvalid) || !strings.Contains(err.Error(), tc.says) {
			t.Errorf("%.80q: %+v, %v; want refused saying %q", tc.link, got, err, tc.says)
		}
	}
}

// FuzzParseDeepLink: no input panics, and what parses keeps every rule: bounded, printable,
// one of the forms, a name only with a kind's view.
func FuzzParseDeepLink(f *testing.F) {
	for _, s := range []string{
		"cogpit://open?context=kind-dev+%5Bconfig%5D&view=pods&namespace=team-a&name=web-0",
		"cogpit://open?view=cluster/overview", "cogpit://open?namespace=a", "cogpit:open", "cogpit://open?a=%",
		"cogpit://open?context=%E2%80%AE", "cogpit://%6Fpen?view=pods", "cogpit://open?view=pods#x",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, raw string) {
		l, err := ParseDeepLink(raw)
		if err != nil {
			if !errors.Is(err, ErrDeepLinkInvalid) {
				t.Fatalf("%q: %v does not wrap ErrDeepLinkInvalid", raw, err)
			}
			if l != (DeepLink{}) {
				t.Fatalf("%q: refused but returned %+v", raw, l)
			}
			return
		}
		if len(raw) > MaxDeepLink || l == (DeepLink{}) {
			t.Fatalf("%q parsed into %+v", raw, l)
		}
		for _, v := range []string{l.Context, l.View, l.Namespace, l.Name} {
			if !utf8.ValidString(v) || strings.IndexFunc(v, func(r rune) bool { return unicode.IsControl(r) || unicode.Is(unicode.Cf, r) }) >= 0 {
				t.Fatalf("%q: a value %q holds control or format characters", raw, v)
			}
		}
		if (l.View != "" && !linkViewRE.MatchString(l.View)) || (l.Namespace != "" && !linkNamespaceRE.MatchString(l.Namespace)) ||
			(l.Name != "" && (!linkNameRE.MatchString(l.Name) || l.View == "" || strings.Contains(l.View, "/"))) ||
			len(l.Context) > maxLinkContext || len(l.Name) > maxLinkName {
			t.Fatalf("%q parsed into %+v", raw, l)
		}
	})
}

// The walls are the command's, not OpenDeepLink's: were a link's command ever to become a write
// (mutating, escalating, declaring a writing verb, GUI-only), Dispatch refuses it to origin
// deeplink before anything runs or anyone is asked (NeedsConfirmation never sees a link's write);
// a navigating one runs.
func TestDeepLinkGuardsAnAlteredCommand(t *testing.T) {
	var asked atomic.Int32
	s := New(Options{Confirmer: ConfirmFunc(func(context.Context, ConfirmRequest) (bool, error) {
		asked.Add(1)
		return true, nil
	})})
	t.Cleanup(s.Close)
	var ran int
	ref := func(_ *Session, p refParams) ([]model.ObjectRef, error) { return []model.ObjectRef{p.Ref}, nil }
	run := func(context.Context, *Session, refParams) (any, error) { ran++; return nil, nil }
	for _, spec := range []Spec[refParams]{
		{ID: "ui.navigate", Label: "x", Danger: DangerMutating, Targets: ref, Run: run},
		{ID: "ui.openView", Label: "x", Danger: DangerUI, Verbs: []Verb{{Verb: "patch", Resource: "*"}}, Run: run},
		{ID: "ui.setNamespace", Label: "x", Danger: DangerUI, GUIOnly: errors.New("the user's"), Run: run},
		{ID: "cluster.connect", Label: "x", Danger: DangerDestructive, Targets: ref, Run: run},
		{ID: "object.delete", Label: "x", Danger: DangerUI, Run: run},
		{ID: "settings.close", Label: "x", Danger: DangerUI, Verbs: []Verb{{Verb: "list", Resource: "*"}}, Run: run},
	} {
		if err := Register(s.Registry(), spec); err != nil {
			t.Fatal(err)
		}
	}
	for _, id := range []string{"ui.navigate", "ui.openView", "ui.setNamespace", "cluster.connect", "object.delete"} {
		_, err := s.Dispatch(context.Background(), Invocation{ID: id, Params: []byte(`{"ref":{"kind":"pods","name":"a"}}`), Origin: OriginDeepLink})
		if !errors.Is(err, ErrDeepLinkRefused) {
			t.Errorf("%s: %v, want refused", id, err)
		}
	}
	if ran != 0 || asked.Load() != 0 {
		t.Errorf("ran %d, asked %d", ran, asked.Load())
	}
	if _, err := s.Dispatch(context.Background(), Invocation{ID: "settings.close", Params: []byte(`{"ref":{"kind":"pods","name":"a"}}`), Origin: OriginDeepLink}); err != nil || ran != 1 {
		t.Errorf("a navigating command: %v, ran %d", err, ran)
	}
	// The second wall: an invocation that escalates is refused after Escalate decided.
	if err := checkDeepLinkDanger("x", OriginDeepLink, DangerDestructive); !errors.Is(err, ErrDeepLinkRefused) {
		t.Errorf("an escalated danger: %v", err)
	}
	if err := checkDeepLinkDanger("x", OriginGUI, DangerDestructive); err != nil {
		t.Errorf("the GUI: %v", err)
	}
}

// The queue holds links until the window attaches, hands them on in their order one at a time, keeps
// at most MaxQueuedDeepLinks waiting, drops an overlong one, waits again once detached, and a
// handler's panic costs that link only.
func TestDeepLinkQueue(t *testing.T) {
	q := NewDeepLinkQueue(nil)
	q.Deliver(strings.Repeat("x", MaxDeepLink+1))
	for i := range MaxQueuedDeepLinks + 3 {
		q.Deliver(fmt.Sprintf("cogpit://open?view=v%d", i))
	}
	if n := q.Pending(); n != MaxQueuedDeepLinks {
		t.Fatalf("%d waiting, want %d", n, MaxQueuedDeepLinks)
	}
	var mu sync.Mutex
	var got []string
	inHandler := atomic.Int32{}
	done := make(chan struct{}, 64)
	detach := q.Attach(func(raw string) {
		if inHandler.Add(1) != 1 {
			t.Error("two links handled at once")
		}
		defer inHandler.Add(-1)
		mu.Lock()
		got = append(got, raw)
		mu.Unlock()
		if strings.HasSuffix(raw, "v2") {
			panic("boom")
		}
		done <- struct{}{}
	})
	waitFor(t, "the queued links", func() bool { mu.Lock(); defer mu.Unlock(); return len(got) == MaxQueuedDeepLinks })
	for i, raw := range got {
		if raw != fmt.Sprintf("cogpit://open?view=v%d", i) {
			t.Fatalf("link %d is %q: out of order (%v)", i, raw, got)
		}
	}
	q.Deliver("cogpit://open?view=later")
	waitFor(t, "the later link", func() bool { mu.Lock(); defer mu.Unlock(); return slices.Contains(got, "cogpit://open?view=later") })
	detach()
	q.Deliver("cogpit://open?view=detached")
	time.Sleep(20 * time.Millisecond)
	if n := q.Pending(); n != 1 {
		t.Errorf("%d waiting after detach, want 1", n)
	}
	var nilQueue *DeepLinkQueue
	nilQueue.Deliver("x")
	nilQueue.Attach(func(string) {})()
}

// Links delivered from many threads at once while the window attaches: none is lost within the
// bound, none handled twice (run with -race).
func TestDeepLinkQueueConcurrent(t *testing.T) {
	q := NewDeepLinkQueue(nil)
	var handled sync.Map
	var count atomic.Int32
	var wg sync.WaitGroup
	for g := range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 2 {
				q.Deliver(fmt.Sprintf("cogpit://open?view=g%di%d", g, i))
			}
		}()
	}
	detach := q.Attach(func(raw string) {
		if _, dup := handled.LoadOrStore(raw, true); dup {
			t.Errorf("%s handled twice", raw)
		}
		count.Add(1)
	})
	defer detach()
	wg.Wait()
	waitFor(t, "every link", func() bool { return count.Load() == 8 })
}
