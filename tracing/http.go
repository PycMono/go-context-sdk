package tracing

import (
	"context"
	"net/http"
	"strings"

	"go.opentelemetry.io/otel/propagation"
)

const (
	// headerTraceparent / headerTracestate 是 W3C Trace Context 规范定义的两个传播头。
	headerTraceparent = "traceparent"
	headerTracestate  = "tracestate"
)

var traceContextPropagator = propagation.TraceContext{}

// Extract returns a context carrying a valid W3C remote parent from h.
func Extract(ctx context.Context, h http.Header) context.Context {
	if h == nil {
		return ctx
	}
	traceparents := h.Values(headerTraceparent)
	if len(traceparents) != 1 || strings.Contains(traceparents[0], ",") {
		return ctx
	}
	carrierHeader := h
	if tracestate := h.Values(headerTracestate); len(tracestate) > 1 {
		carrierHeader = h.Clone()
		carrierHeader.Set(headerTracestate, strings.Join(tracestate, ","))
	}
	return traceContextPropagator.Extract(ctx, propagation.HeaderCarrier(carrierHeader))
}

// Inject writes the W3C trace context from ctx into h.
func Inject(ctx context.Context, h http.Header) {
	if h == nil {
		return
	}
	traceContextPropagator.Inject(ctx, propagation.HeaderCarrier(h))
}
