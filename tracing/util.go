package tracing

import (
	"context"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

// NewRequestID 生成一个新的 requestID（标准 UUID v4）
func NewRequestID() string {
	return uuid.New().String()
}

// TraceIDFromContext returns the active OTel trace ID, or an empty string.
func TraceIDFromContext(ctx context.Context) string {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return ""
	}
	return spanContext.TraceID().String()
}
