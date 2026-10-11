package cluster

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const endpointsDeprecation = "v1 Endpoints is deprecated in v1.33+; use discovery.k8s.io/v1 EndpointSlice"

// TestWarningLogOncePerMessage: each distinct warning is logged once, cut when long; repeats,
// warnings beyond the bound and other codes are counted or dropped, never logged again.
func TestWarningLogOncePerMessage(t *testing.T) {
	var buf bytes.Buffer
	w := newWarningLog("kind-test")
	w.log = slog.New(slog.NewTextHandler(&buf, nil))
	for range 5 {
		w.HandleWarningHeaderWithContext(context.Background(), 299, "-", endpointsDeprecation)
	}
	w.HandleWarningHeaderWithContext(context.Background(), 199, "-", "not a 299 warning")
	w.HandleWarningHeaderWithContext(context.Background(), 299, "-", "")
	long := strings.Repeat("x", 2*maxWarningLen)
	w.HandleWarningHeaderWithContext(context.Background(), 299, "-", long)
	if n := strings.Count(buf.String(), "level=WARN"); n != 2 {
		t.Errorf("%d lines logged, want 2:\n%s", n, buf.String())
	}
	if strings.Contains(buf.String(), long) || !strings.Contains(buf.String(), "context=kind-test") {
		t.Errorf("log %s", buf.String())
	}
	distinct, received := w.warnings()
	if !slices.Equal(distinct, []string{endpointsDeprecation, long}) || received != 6 {
		t.Errorf("warnings %q, %d received", distinct, received)
	}
	for i := range 2 * maxWarnings {
		w.HandleWarningHeaderWithContext(context.Background(), 299, "-", fmt.Sprint("warning ", i))
	}
	if distinct, _ := w.warnings(); len(distinct) != maxWarnings {
		t.Errorf("%d distinct warnings kept, want at most %d", len(distinct), maxWarnings)
	}
}

// TestConnectHandlesWarnings: a connection's clients hand the API server's warnings to the
// connection's warning log, so that a deprecated kind's warning, which comes with every request,
// is kept once.
func TestConnectHandlesWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/version" {
			fmt.Fprint(w, `{"gitVersion":"v1.36.1"}`)
			return
		}
		w.Header().Add("Warning", `299 - "`+endpointsDeprecation+`"`)
		fmt.Fprint(w, `{"apiVersion":"v1","kind":"EndpointsList","metadata":{"resourceVersion":"7"},"items":[]}`)
	}))
	defer srv.Close()
	path := writeFile(t, "config", kubeconfigFor("warned", srv.URL, ""))
	c, err := Connect(context.Background(), Options{KubeconfigPaths: []string{path}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	endpoints := schema.GroupVersionResource{Version: "v1", Resource: "endpoints"}
	for range 3 {
		if _, err := c.Dynamic().Resource(endpoints).Namespace("shop").List(context.Background(), metav1.ListOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	if distinct, received := c.Warnings(); !slices.Equal(distinct, []string{endpointsDeprecation}) || received != 3 {
		t.Errorf("warnings %q, %d received; want the deprecation once of 3", distinct, received)
	}
	if c.RESTConfig().WarningHandlerWithContext == nil {
		t.Error("a copy of the REST config lost the warning handler")
	}
}
