package httpserver

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanHandlerContinuesIncomingTraceAndInstrumentsOutboundHTTP(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
		_ = provider.Shutdown(context.Background())
	})

	var downstreamTraceparent string
	var server *server
	downstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		downstreamTraceparent = r.Header.Get("traceparent")
		server.Redirect(w, r)
	}))
	t.Cleanup(downstream.Close)

	var err error
	server, err = NewServer(downstream.URL, "", "")
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/span", nil)
	request.Header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	response := httptest.NewRecorder()

	server.SpanHandler(response, request)

	var entryServerSpan, downstreamServerSpan, clientSpan sdktrace.ReadOnlySpan
	for _, span := range recorder.Ended() {
		switch span.SpanKind() {
		case trace.SpanKindServer:
			if span.Name() == "span-test-parent" {
				entryServerSpan = span
			} else if span.Name() == "span-test-op2" {
				downstreamServerSpan = span
			}
		case trace.SpanKindClient:
			clientSpan = span
		}
	}
	if entryServerSpan == nil {
		t.Fatalf("SERVER span missing: %#v", recorder.Ended())
	}
	if got := entryServerSpan.SpanContext().TraceID().String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("SERVER trace ID = %s", got)
	}
	if !entryServerSpan.Parent().IsRemote() || entryServerSpan.Parent().SpanID().String() != "00f067aa0ba902b7" {
		t.Fatalf("SERVER parent = %v, want remote parent", entryServerSpan.Parent())
	}
	if clientSpan == nil {
		t.Fatalf("CLIENT span missing: %#v", recorder.Ended())
	}
	if clientSpan.SpanContext().TraceID() != entryServerSpan.SpanContext().TraceID() {
		t.Fatalf("CLIENT trace ID = %s, SERVER trace ID = %s", clientSpan.SpanContext().TraceID(), entryServerSpan.SpanContext().TraceID())
	}
	if downstreamServerSpan == nil {
		t.Fatalf("downstream SERVER span missing: %#v", recorder.Ended())
	}
	if downstreamServerSpan.Parent().SpanID() != clientSpan.SpanContext().SpanID() {
		t.Fatalf("downstream parent = %s, CLIENT span ID = %s", downstreamServerSpan.Parent().SpanID(), clientSpan.SpanContext().SpanID())
	}
	if !strings.Contains(downstreamTraceparent, "4bf92f3577b34da6a3ce929d0e0e4736") {
		t.Fatalf("downstream traceparent = %q", downstreamTraceparent)
	}
}

type trackingBody struct {
	io.Reader
	closed bool
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func TestSpanHandlerClosesDownstreamResponseBody(t *testing.T) {
	server, err := NewServer("http://downstream.test", "", "")
	if err != nil {
		t.Fatal(err)
	}
	body := &trackingBody{Reader: strings.NewReader("ok")}
	server.client = &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       body,
		}, nil
	})}

	server.SpanHandler(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/span", nil))

	if !body.closed {
		t.Fatal("downstream response body was not closed")
	}
}
