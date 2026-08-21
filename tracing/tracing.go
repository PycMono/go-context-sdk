package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const (
	instrumentationScopeName    = "github.com/PycMono/go-context-sdk"
	instrumentationScopeVersion = "v1.2.0"
)

// StartSpan starts a span using the application's global OTel TracerProvider.
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	tracer := otel.Tracer(
		instrumentationScopeName,
		trace.WithInstrumentationVersion(instrumentationScopeVersion),
	)
	return tracer.Start(ctx, name, opts...)
}
