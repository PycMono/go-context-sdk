package tracing

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

func installRecordingProvider(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	t.Helper()
	previousProvider := otel.GetTracerProvider()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	otel.SetTracerProvider(provider)
	t.Cleanup(func() {
		otel.SetTracerProvider(previousProvider)
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown tracer provider: %v", err)
		}
	})
	return provider, recorder
}

func installNoopProvider(t *testing.T) {
	t.Helper()
	previousProvider := otel.GetTracerProvider()
	otel.SetTracerProvider(noop.NewTracerProvider())
	t.Cleanup(func() { otel.SetTracerProvider(previousProvider) })
}

func TestStartSpanUsesGlobalProviderAndInstrumentationScope(t *testing.T) {
	_, recorder := installRecordingProvider(t)

	ctx, span := StartSpan(context.Background(), "root")
	if ctx == nil || span == nil {
		t.Fatalf("invalid start result: ctx=%v span=%v", ctx, span)
	}
	got := trace.SpanFromContext(ctx).SpanContext()
	want := span.SpanContext()
	if got.TraceID() != want.TraceID() || got.SpanID() != want.SpanID() {
		t.Fatalf("context span = %v, returned span = %v", got, want)
	}
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 || ended[0].Name() != "root" {
		t.Fatalf("ended spans = %#v, want one root span", ended)
	}
	if ended[0].Parent().IsValid() {
		t.Fatalf("root parent = %v, want invalid", ended[0].Parent())
	}
	scope := ended[0].InstrumentationScope()
	if scope.Name != instrumentationScopeName || scope.Version != instrumentationScopeVersion {
		t.Fatalf("scope = %#v, want name=%q version=%q", scope, instrumentationScopeName, instrumentationScopeVersion)
	}
}

func TestStartSpanPreservesParentFromGlobalProvider(t *testing.T) {
	provider, recorder := installRecordingProvider(t)
	parentCtx, parent := provider.Tracer("test-parent").Start(context.Background(), "parent")

	_, child := StartSpan(parentCtx, "child")
	child.End()
	parent.End()

	ended := recorder.Ended()
	if len(ended) != 2 {
		t.Fatalf("ended span count = %d, want 2", len(ended))
	}
	childSpan := ended[0]
	if childSpan.SpanContext().TraceID() != parent.SpanContext().TraceID() {
		t.Fatalf("child trace ID = %s, want %s", childSpan.SpanContext().TraceID(), parent.SpanContext().TraceID())
	}
	if childSpan.Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Fatalf("child parent = %s, want %s", childSpan.Parent().SpanID(), parent.SpanContext().SpanID())
	}
}

func TestStartSpanWithNoopDoesNotGenerateRootTraceID(t *testing.T) {
	installNoopProvider(t)

	ctx, span := StartSpan(context.Background(), "noop-root")
	defer span.End()

	if span.IsRecording() {
		t.Fatal("noop span should not record")
	}
	if got := trace.SpanContextFromContext(ctx); got.IsValid() {
		t.Fatalf("noop root span context = %v, want invalid", got)
	}
	if got := TraceIDFromContext(ctx); got != "" {
		t.Fatalf("noop root trace ID = %q, want empty", got)
	}
}

func TestStartSpanWithNoopPreservesValidParentTraceID(t *testing.T) {
	installNoopProvider(t)
	parent := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    trace.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16},
		SpanID:     trace.SpanID{1, 2, 3, 4, 5, 6, 7, 8},
		TraceFlags: trace.FlagsSampled,
		Remote:     true,
	})
	parentCtx := trace.ContextWithRemoteSpanContext(context.Background(), parent)

	ctx, span := StartSpan(parentCtx, "noop-child")
	defer span.End()

	if got := trace.SpanContextFromContext(ctx).TraceID(); got != parent.TraceID() {
		t.Fatalf("noop child trace ID = %s, want %s", got, parent.TraceID())
	}
	if got := TraceIDFromContext(ctx); got != parent.TraceID().String() {
		t.Fatalf("TraceIDFromContext = %q, want %q", got, parent.TraceID())
	}
}
