package tracing

import (
	"context"
	"fmt"
	"math"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Span wraps an OTel span with the SDK's convenience methods.
type Span struct {
	trace.Span
}

// StartChildSpan starts a child of s using the configured TracerProvider.
func (s *Span) StartChildSpan(name string) *Span {
	ctx := context.Background()
	if s != nil && s.Span != nil {
		ctx = trace.ContextWithSpan(ctx, s.Span)
	}
	_, child := tracer().Start(ctx, name)
	return &Span{Span: child}
}

// SetTag records a value as an OTel span attribute.
func (s *Span) SetTag(key string, value interface{}) *Span {
	if s == nil || s.Span == nil {
		return s
	}
	s.SetAttributes(tagAttribute(key, value))
	return s
}

func tagAttribute(key string, value interface{}) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int64(key, int64(v))
	case int8:
		return attribute.Int64(key, int64(v))
	case int16:
		return attribute.Int64(key, int64(v))
	case int32:
		return attribute.Int64(key, int64(v))
	case int64:
		return attribute.Int64(key, v)
	case uint:
		return unsignedAttribute(key, uint64(v))
	case uint8:
		return attribute.Int64(key, int64(v))
	case uint16:
		return attribute.Int64(key, int64(v))
	case uint32:
		return attribute.Int64(key, int64(v))
	case uint64:
		return unsignedAttribute(key, v)
	case float64:
		return attribute.Float64(key, v)
	case error:
		return attribute.String(key, v.Error())
	default:
		return attribute.String(key, fmt.Sprint(v))
	}
}

func unsignedAttribute(key string, value uint64) attribute.KeyValue {
	if value <= math.MaxInt64 {
		return attribute.Int64(key, int64(value))
	}
	return attribute.String(key, fmt.Sprint(value))
}

// LogFields records a log event on the span.
func (s *Span) LogFields(attrs ...attribute.KeyValue) {
	if s == nil || s.Span == nil {
		return
	}
	s.AddEvent("log", trace.WithAttributes(attrs...))
}

// Finish ends the span.
func (s *Span) Finish() {
	if s == nil || s.Span == nil {
		return
	}
	s.End()
}
