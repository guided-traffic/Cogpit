package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
)

// statusServer answers every request with a Status of the given code, as the API server answers a
// portforward request it refuses (the WebSocket upgrade fails first, then the SPDY dial gets this).
func statusServer(t *testing.T, code int32, reason metav1.StatusReason, message string) (*rest.Config, *url.URL) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(int(code))
		_ = json.NewEncoder(w).Encode(metav1.Status{
			TypeMeta: metav1.TypeMeta{Kind: "Status", APIVersion: "v1"},
			Status:   metav1.StatusFailure, Code: code, Reason: reason, Message: message,
		})
	}))
	t.Cleanup(srv.Close)
	u, err := url.Parse(srv.URL + "/api/v1/namespaces/shop/pods/web-0/portforward")
	if err != nil {
		t.Fatal(err)
	}
	return &rest.Config{Host: srv.URL}, u
}

// TestDialErrorsAreClassified (D-20): what the real dialer returns for the API server's answers,
// classified as permanent (fail at once) or transient (retry).
func TestDialErrorsAreClassified(t *testing.T) {
	tests := []struct {
		code      int32
		reason    metav1.StatusReason
		permanent bool
	}{
		{http.StatusNotFound, metav1.StatusReasonNotFound, true},
		{http.StatusForbidden, metav1.StatusReasonForbidden, true},
		{http.StatusUnauthorized, metav1.StatusReasonUnauthorized, true},
		{http.StatusBadRequest, metav1.StatusReasonBadRequest, false}, // "pod is not running": passes
		{http.StatusServiceUnavailable, metav1.StatusReasonServiceUnavailable, false},
		{http.StatusInternalServerError, metav1.StatusReasonInternalError, false},
		{http.StatusTooManyRequests, metav1.StatusReasonTooManyRequests, false},
		{http.StatusGatewayTimeout, metav1.StatusReasonTimeout, false},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprint(tt.code), func(t *testing.T) {
			cfg, u := statusServer(t, tt.code, tt.reason, "the server says no")
			_, err := dialPortForward(context.Background(), cfg, u)
			if err == nil {
				t.Fatal("the dial succeeded")
			}
			if got := IsPermanentForwardError(err); got != tt.permanent {
				t.Errorf("IsPermanentForwardError(%v) = %v, want %v", err, got, tt.permanent)
			}
			// Through DialPortForward's wrapping as the engine sees it.
			if got := IsPermanentForwardError(fmt.Errorf("port forward to shop/web-0: %w", err)); got != tt.permanent {
				t.Errorf("wrapped: %v, want %v", got, tt.permanent)
			}
		})
	}
	for _, err := range []error{nil, errors.New("connection refused"), context.DeadlineExceeded, context.Canceled, errors.New("unable to upgrade connection: Forbidden")} {
		if IsPermanentForwardError(err) {
			t.Errorf("%v is not a typed refusal and is retried", err)
		}
	}
}
