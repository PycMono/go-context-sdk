package tracing

import (
	"context"
	"os"
	"sync"
	"testing"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace/noop"
)

func TestServiceNameUsesOTelPriority(t *testing.T) {
	t.Setenv("OTEL_SERVICE_NAME", "otel-service")
	t.Setenv("SERVICE_NAME", "service-name")
	t.Setenv("JAEGER_SERVICE_NAME", "legacy-jaeger")
	if got := ServiceName(); got != "otel-service" {
		t.Fatalf("service name = %q, want otel-service", got)
	}

	t.Setenv("OTEL_SERVICE_NAME", "")
	if got := ServiceName(); got != "service-name" {
		t.Fatalf("service name = %q, want service-name", got)
	}

	t.Setenv("SERVICE_NAME", "")
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatal(err)
	}
	if got := ServiceName(); got != hostname {
		t.Fatalf("service name = %q, want hostname %q", got, hostname)
	}
}

func newRecordingProvider(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			t.Errorf("shutdown tracer provider: %v", err)
		}
	})
	return tp, recorder
}

func TestStartSpanFromContextUsesNoopProvider(t *testing.T) {
	Init(noop.NewTracerProvider())

	ctx, span := StartSpanFromContext(context.Background(), "noop")

	if ctx == nil || span == nil || span.Span == nil {
		t.Fatalf("invalid start result: ctx=%v span=%#v", ctx, span)
	}
	if span.IsRecording() {
		t.Fatal("span should not record with the noop provider")
	}
}

func TestInitUsesProvidedTracerProvider(t *testing.T) {
	tp, recorder := newRecordingProvider(t)
	Init(tp)

	_, span := StartSpanFromContext(context.Background(), "root")
	span.Finish()

	ended := recorder.Ended()
	if len(ended) != 1 || ended[0].Name() != "root" {
		t.Fatalf("ended spans = %#v, want one root span", ended)
	}
}

func TestStartSpanFromContextPreservesParent(t *testing.T) {
	tp, recorder := newRecordingProvider(t)
	Init(tp)
	parentCtx, parent := tp.Tracer("test-parent").Start(context.Background(), "parent")

	_, child := StartSpanFromContext(parentCtx, "child")
	child.Finish()
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

func TestInitNilKeepsProvider(t *testing.T) {
	tp, recorder := newRecordingProvider(t)
	Init(tp)
	Init(nil)

	_, span := StartSpanFromContext(context.Background(), "after-nil")
	span.Finish()

	if got := len(recorder.Ended()); got != 1 {
		t.Fatalf("ended span count = %d, want 1", got)
	}
}

func TestInitConcurrentWithStartSpan(t *testing.T) {
	first, firstRecorder := newRecordingProvider(t)
	second, secondRecorder := newRecordingProvider(t)
	Init(first)

	const iterations = 1000
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			if i%2 == 0 {
				Init(first)
			} else {
				Init(second)
			}
		}
	}()

	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < iterations; i++ {
			_, span := StartSpanFromContext(context.Background(), "concurrent")
			span.Finish()
		}
	}()

	close(start)
	wg.Wait()

	if got := len(firstRecorder.Ended()) + len(secondRecorder.Ended()); got != iterations {
		t.Fatalf("ended span count = %d, want %d", got, iterations)
	}
}
