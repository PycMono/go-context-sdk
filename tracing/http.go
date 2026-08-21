package tracing

import (
	"context"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

var traceContextPropagator = propagation.TraceContext{}

// Extract returns a context carrying a valid W3C remote parent from h.
func Extract(ctx context.Context, h http.Header) context.Context {
	if h == nil {
		return ctx
	}
	traceparents := h.Values("traceparent")
	if len(traceparents) != 1 || strings.Contains(traceparents[0], ",") {
		return ctx
	}
	return traceContextPropagator.Extract(ctx, propagation.HeaderCarrier(h))
}

// Inject writes the W3C trace context from ctx into h.
func Inject(ctx context.Context, h http.Header) {
	if h == nil {
		return
	}
	traceContextPropagator.Inject(ctx, propagation.HeaderCarrier(h))
}
