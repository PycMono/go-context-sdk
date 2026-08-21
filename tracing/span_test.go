package tracing

import (
	"context"
	"errors"
	"math"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestSpanStartChildSpanUsesParent(t *testing.T) {
	tp, recorder := newRecordingProvider(t)
	Init(tp)
	_, parent := StartSpanFromContext(context.Background(), "parent")

	child := parent.StartChildSpan("child")
	child.Finish()
	parent.Finish()

	ended := recorder.Ended()
	if len(ended) != 2 {
		t.Fatalf("ended span count = %d, want 2", len(ended))
	}
	if ended[0].Parent().SpanID() != parent.SpanContext().SpanID() {
		t.Fatalf("child parent = %s, want %s", ended[0].Parent().SpanID(), parent.SpanContext().SpanID())
	}
}

func TestSpanSetTagConvertsValues(t *testing.T) {
	tp, recorder := newRecordingProvider(t)
	Init(tp)
	_, span := StartSpanFromContext(context.Background(), "tags")

	span.SetTag("string", "value")
	span.SetTag("bool", true)
	span.SetTag("int", int(-1))
	span.SetTag("int8", int8(-8))
	span.SetTag("int16", int16(-16))
	span.SetTag("int32", int32(-32))
	span.SetTag("int64", int64(-64))
	span.SetTag("uint", uint(1))
	span.SetTag("uint8", uint8(8))
	span.SetTag("uint16", uint16(16))
	span.SetTag("uint32", uint32(32))
	span.SetTag("uint64-max-int", uint64(math.MaxInt64))
	span.SetTag("uint64-overflow", uint64(math.MaxInt64)+1)
	span.SetTag("float64", 1.25)
	span.SetTag("error", errors.New("boom"))
	span.SetTag("other", struct{ N int }{N: 7})
	span.Finish()

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended span count = %d, want 1", len(ended))
	}
	got := make(map[string]interface{})
	for _, attr := range ended[0].Attributes() {
		got[string(attr.Key)] = attr.Value.AsInterface()
	}
	want := map[string]interface{}{
		"string":          "value",
		"bool":            true,
		"int":             int64(-1),
		"int8":            int64(-8),
		"int16":           int64(-16),
		"int32":           int64(-32),
		"int64":           int64(-64),
		"uint":            int64(1),
		"uint8":           int64(8),
		"uint16":          int64(16),
		"uint32":          int64(32),
		"uint64-max-int":  int64(math.MaxInt64),
		"uint64-overflow": "9223372036854775808",
		"float64":         float64(1.25),
		"error":           "boom",
		"other":           "{7}",
	}
	for key, expected := range want {
		if actual := got[key]; actual != expected {
			t.Errorf("attribute %q = %#v, want %#v", key, actual, expected)
		}
	}
}

func TestSpanLogFieldsAddsLogEvent(t *testing.T) {
	tp, recorder := newRecordingProvider(t)
	Init(tp)
	_, span := StartSpanFromContext(context.Background(), "events")

	span.LogFields(attribute.String("message", "hello"), attribute.Int("attempt", 2))
	span.Finish()

	events := recorder.Ended()[0].Events()
	if len(events) != 1 || events[0].Name != "log" {
		t.Fatalf("events = %#v, want one log event", events)
	}
	if len(events[0].Attributes) != 2 {
		t.Fatalf("event attributes = %#v, want 2", events[0].Attributes)
	}
}

func TestSpanFromContextAndTraceID(t *testing.T) {
	tp, _ := newRecordingProvider(t)
	Init(tp)
	ctx, started := StartSpanFromContext(context.Background(), "lookup")
	defer started.Finish()

	found := SpanFromContext(ctx)
	if found == nil ||
		found.SpanContext().TraceID() != started.SpanContext().TraceID() ||
		found.SpanContext().SpanID() != started.SpanContext().SpanID() {
		t.Fatalf("found span context = %v, want %v", found, started.SpanContext())
	}
	if got := TraceIDFromContext(ctx); got != started.SpanContext().TraceID().String() {
		t.Fatalf("trace ID = %q, want %q", got, started.SpanContext().TraceID())
	}
	if got := TraceIDFromContext(context.Background()); got != "" {
		t.Fatalf("trace ID without span = %q, want empty", got)
	}
}
