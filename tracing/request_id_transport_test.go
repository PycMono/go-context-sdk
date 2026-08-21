package tracing

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PycMono/go-context-sdk/bizctx"
)

type recordingRoundTripper struct {
	request *http.Request
}

func (r *recordingRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	r.request = request
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader("ok")),
		Request:    request,
	}, nil
}

func TestRequestIDTransportUsesDefaultTransportForNilBase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get(HeaderRequestID); got != "request-1" {
			t.Errorf("request-id = %q, want request-1", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	request, err := http.NewRequestWithContext(
		bizctx.WithKV(context.Background(), bizctx.RequestID("request-1")),
		http.MethodGet,
		server.URL,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	response, err := (&http.Client{Transport: NewRequestIDTransport(nil)}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
}

func TestRequestIDTransportPassesThroughWithoutContextID(t *testing.T) {
	base := &recordingRoundTripper{}
	transport := NewRequestIDTransport(base)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set(HeaderRequestID, "existing")

	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if base.request != request {
		t.Fatal("request without context ID should pass through unchanged")
	}
	if got := request.Header.Get(HeaderRequestID); got != "existing" {
		t.Fatalf("original request-id = %q", got)
	}
}

func TestRequestIDTransportClonesAndOverwritesHeader(t *testing.T) {
	base := &recordingRoundTripper{}
	transport := NewRequestIDTransport(base)
	type contextKey struct{}
	ctx := context.WithValue(context.Background(), contextKey{}, "kept")
	ctx = bizctx.WithKV(ctx, bizctx.RequestID("from-context"))
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set(HeaderRequestID, "existing")
	request.Header.Set("X-Keep", "value")

	response, err := transport.RoundTrip(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	if base.request == request {
		t.Fatal("request with context ID was not cloned")
	}
	if got := base.request.Header.Get(HeaderRequestID); got != "from-context" {
		t.Fatalf("outbound request-id = %q", got)
	}
	if got := base.request.Header.Get("X-Keep"); got != "value" {
		t.Fatalf("cloned header lost value: %q", got)
	}
	if got := base.request.Context().Value(contextKey{}); got != "kept" {
		t.Fatalf("cloned request context value = %#v", got)
	}
	if got := request.Header.Get(HeaderRequestID); got != "existing" {
		t.Fatalf("original request header mutated: %q", got)
	}
}
