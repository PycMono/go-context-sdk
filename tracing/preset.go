package tracing

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"net/http"
	"strings"
)

type contextKey struct{}

var (
	activeSpanKey = contextKey{}
)

// Inject 将Span信息注入到 `http.Request` 中
func Inject(r *http.Request, span *Span) error {
	if r == nil {
		return ErrNullRequest
	}

	if span == nil {
		return nil
	}
	for k, v := range span.SpanContext {
		r.Header.Set(k, v)
	}
	return nil
}

//// Inject2Grpc 将Span信息注入到 `grpc` 中
//func Inject2Grpc(bizctx context.Context, span Span) context.Context {
//	kv := []string{}
//	for k, v := range span.FilterOutSpan() {
//		kv = append(kv, k, v)
//	}
//	return metadata.AppendToOutgoingContext(bizctx, kv...)
//}

// Extract 从 `http.Request` 提取Span信息
func Extract(r *http.Request, extHeaders ...string) (*Span, error) {
	if r == nil {
		return nil, ErrNullRequest
	}

	// 获取请求头中的span信息
	if span := SpanFromContext(r.Context()); !span.Empty() {
		return span, nil
	}

	span := extract(r.Header, extHeaders...)
	span.Set(HeaderRequestPath, r.URL.Path)
	return span, nil
}

// StartNewSpan 创建一个新的Span
func StartNewSpan(operationName string, opts ...opentracing.StartSpanOption) *Span {
	span := opentracing.StartSpan(operationName, opts...)
	// 设置traceID
	spanCtx, ok := span.Context().(jaeger.SpanContext)
	if !ok {
		return &Span{
			Span:        span,
			SpanContext: make(SpanContext),
		}
	}

	span.SetBaggageItem("x-b3-traceid", spanCtx.TraceID().String())
	if req := span.BaggageItem(TagRequestID); len(req) == 0 {
		span.SetBaggageItem(TagRequestID, NewRequestID())
	}
	return &Span{
		Span:        span,
		SpanContext: make(SpanContext),
	}
}

// SpanFromContext returns the `Span` previously associated with `bizctx`, or
// `empty map` if no such `Span` could be found.
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
	// todo 完善grpc
	return out
}

// SpanFromOpentracing returns the `Span` previously associated with `bizctx`, or
func SpanFromOpentracing(span opentracing.Span) *Span {
	out := &Span{SpanContext: make(SpanContext)}
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

func extract(headers map[string][]string, extHeaders ...string) *Span {
	out := &Span{SpanContext: make(SpanContext)}
	for rawKey, rawValue := range headers {
		var value string
		if len(rawValue) > 0 {
			value = rawValue[0]
		}
		key := strings.ToLower(rawKey)
		out.Set(key, value) // todo 这里会有大量无用的数据，需要做过滤
	}
	return out
}
