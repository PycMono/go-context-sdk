package tracing

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

// StartSpanFromContext starts a span and returns the context carrying it.
func StartSpanFromContext(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, *Span) {
	ctx, span := tracer().Start(ctx, name, opts...)
	return ctx, &Span{Span: span}
}

// SpanFromContext returns the OTel span carried by ctx.
func SpanFromContext(ctx context.Context) *Span {
	return &Span{Span: trace.SpanFromContext(ctx)}
}
