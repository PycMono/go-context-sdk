package ctxsdk

import (
	"github.com/google/uuid"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

// BizCtxValue value
type BizCtxValue map[string]string

// Set 设置
func (c BizCtxValue) Set(k, v string) {
	c[k] = v
}

// SetTraceID ...
func (c BizCtxValue) SetTraceID() {
	c.Set(traceID, NewID())
}

// IsEmpty 是否为空
func (c BizCtxValue) IsEmpty() bool {
	return len(c) == 0
}

// BizCtxValue4Span 从spa获取BizCtxValue
func BizCtxValue4Span(span opentracing.Span) BizCtxValue {
	spanCtx, ok := span.Context().(jaeger.SpanContext)
	if !ok {
		return BizCtxValue{}
	}

	out := BizCtxValue{}
	out.Set(traceID, spanCtx.TraceID().String())
	out.Set(parentSpanID, spanCtx.ParentID().String())
	out.Set(spanID, spanCtx.SpanID().String())

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

// NewID ...
func NewID() string {
	u := uuid.New().String()
	return u[0:14] + "9" + u[15:]
}
