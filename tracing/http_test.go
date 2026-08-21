package tracing

import (
	"context"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/trace"
)

func TestExtractValidTraceContext(t *testing.T) {
	header := make(http.Header)
	header.Set("traceparent", "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	header.Set("tracestate", "vendor=value")

	ctx := Extract(context.Background(), header)
	sc := trace.SpanContextFromContext(ctx)

	if !sc.IsValid() || !sc.IsRemote() {
		t.Fatalf("extracted span context = %v, want valid remote", sc)
	}
	if got := sc.TraceID().String(); got != "4bf92f3577b34da6a3ce929d0e0e4736" {
		t.Fatalf("trace ID = %s", got)
	}
	if got := sc.SpanID().String(); got != "00f067aa0ba902b7" {
		t.Fatalf("span ID = %s", got)
	}
	if got := sc.TraceState().String(); got != "vendor=value" {
		t.Fatalf("tracestate = %q", got)
	}
}

func TestExtractIgnoresInvalidTraceparent(t *testing.T) {
	tests := []struct {
		name   string
		header http.Header
	}{
		{name: "nil header", header: nil},
		{name: "missing", header: make(http.Header)},
		{name: "malformed", header: http.Header{"Traceparent": []string{"invalid"}}},
		{name: "multiple", header: http.Header{"Traceparent": []string{
			"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01",
			"00-4bf92f3577b34da6a3ce929d0e0e4736-11f067aa0ba902b7-01",
		}}},
		{name: "comma joined", header: http.Header{"Traceparent": []string{
			"00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01,00-4bf92f3577b34da6a3ce929d0e0e4736-11f067aa0ba902b7-01",
		}}},
	}

	type contextKey struct{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := context.WithValue(context.Background(), contextKey{}, "kept")
			got := Extract(base, tt.header)
			if got.Value(contextKey{}) != "kept" {
				t.Fatal("base context value was lost")
			}
			if sc := trace.SpanContextFromContext(got); sc.IsValid() {
				t.Fatalf("unexpected span context: %v", sc)
			}
		})
	}
}

func TestInjectWritesTraceContext(t *testing.T) {
	traceState, err := trace.ParseTraceState("vendor=value")
	if err != nil {
		t.Fatal(err)
	}
	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
		TraceState: traceState,
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	header := make(http.Header)

	Inject(ctx, header)

	if got := header.Get("traceparent"); got != "00-0102030405060708090a0b0c0d0e0f10-0102030405060708-01" {
		t.Fatalf("traceparent = %q", got)
	}
	if got := header.Get("tracestate"); got != "vendor=value" {
		t.Fatalf("tracestate = %q", got)
	}
}

func TestInjectNilHeaderIsNoop(t *testing.T) {
	Inject(context.Background(), nil)
}
