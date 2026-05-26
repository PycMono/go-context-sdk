package tracing

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

type contextKey struct{}

var (
	activeSpanKey = contextKey{}
)

// StartSpanFromContext 从 context 中恢复 parent span 并创建子 span。
// 如果 context 中没有有效的 parent span，则创建一个新的 root span。
func StartSpanFromContext(ctx context.Context, operationName string, opts ...opentracing.StartSpanOption) *Span {
	if parent := SpanFromContext(ctx); parent != nil && !parent.Empty() {
		return parent.StartChildSpan(operationName, opts...)
	}
	return StartNewSpan(operationName, opts...)
}

// StartNewSpan 创建一个新的Span
func StartNewSpan(operationName string, opts ...opentracing.StartSpanOption) *Span {
	span := opentracing.StartSpan(operationName, opts...)

	out := &Span{
		Span:        span,
		SpanContext: make(SpanContext),
	}

	// 设置traceID
	spanCtx, ok := span.Context().(jaeger.SpanContext)
	if ok {
		out.Set("x-b3-traceid", spanCtx.TraceID().String())
		out.Set("x-b3-spanid", spanCtx.SpanID().String())
		if spanCtx.IsSampled() {
			out.Set("x-b3-sampled", "1")
		}
	}

	// 设置或生成 requestID
	reqID := span.BaggageItem(TagRequestID)
	if len(reqID) == 0 {
		reqID = NewRequestID()
		span.SetBaggageItem(TagRequestID, reqID)
	}
	out.Set(HeaderRequestID, reqID)

	return out
}

// SpanFromContext returns the `Span` previously associated with `ctx`, or
// empty `Span` if no such `Span` could be found.
func SpanFromContext(ctx context.Context) *Span {
	out := &Span{SpanContext: make(SpanContext)}
	// 获取span context
	val := ctx.Value(activeSpanKey)
	if sp, ok := val.(SpanContext); ok {
		for k, v := range sp {
			out.Set(k, v)
		}
		return out
	}

	if span := opentracing.SpanFromContext(ctx); span != nil {
		return SpanFromOpentracing(span)
	}
	return out
}

// SpanFromOpentracing returns the `Span` from opentracing.Span
func SpanFromOpentracing(span opentracing.Span) *Span {
	out := &Span{Span: span, SpanContext: make(SpanContext)}
	spanCtx, ok := span.Context().(jaeger.SpanContext)
	if !ok {
		return out
	}

	out.Set("x-b3-traceid", spanCtx.TraceID().String())
	out.Set("x-b3-parentspanid", spanCtx.ParentID().String())
	out.Set("x-b3-spanid", spanCtx.SpanID().String())
	if spanCtx.IsSampled() {
		out.Set("x-b3-sampled", "1")
	}
	spanCtx.ForeachBaggageItem(func(k, v string) bool {
		if k == TagRequestID {
			out.Set(HeaderRequestID, v)
		} else {
			out.Set(k, v)
		}
		return true
	})

	return out
}
