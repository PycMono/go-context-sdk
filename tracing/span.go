package tracing

import (
	"context"

	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"github.com/uber/jaeger-client-go"
)

// Span is a wrapper of opentracing.Span
type Span struct {
	opentracing.Span
	SpanContext
}

// StartChildSpan start a child span
func (s *Span) StartChildSpan(operationName string, opts ...opentracing.StartSpanOption) *Span {
	if s == nil || s.Empty() {
		return StartNewSpan(operationName, opts...)
	}

	parent, err := s.SwapOpentracingSpanContext()
	if err == nil {
		opts = append(opts, opentracing.ChildOf(parent))
	}

	// 设置基础数据
	span := opentracing.StartSpan(operationName, opts...)
	accountID := s.AccountID()
	if len(accountID) > 0 {
		span.SetBaggageItem(HeaderAuthAccountID, accountID)
	}
	reqID := s.RequestID()
	if len(reqID) > 0 {
		span.SetBaggageItem(TagRequestID, reqID)
	}

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

	// requestID
	if len(reqID) > 0 {
		out.Set(HeaderRequestID, reqID)
	} else if req := span.BaggageItem(TagRequestID); len(req) > 0 {
		out.Set(HeaderRequestID, req)
	} else {
		newReqID := NewRequestID()
		span.SetBaggageItem(TagRequestID, newReqID)
		out.Set(HeaderRequestID, newReqID)
	}

	return out
}

// SetTag set a tag to span
func (s *Span) SetTag(k string, v interface{}) *Span {
	if s != nil && s.Span != nil {
		s.Span.SetTag(k, v)
	}
	return s
}

// WithOpentracingContext return a context with span
func (s *Span) WithOpentracingContext(ctx context.Context) context.Context {
	if s == nil || s.Span == nil {
		return ctx
	}
	return opentracing.ContextWithSpan(ctx, s.Span)
}

// WithContext 设置context value
func (s *Span) WithContext(ctx context.Context) context.Context {
	if s == nil || s.SpanContext == nil {
		return ctx
	}
	return context.WithValue(ctx, activeSpanKey, s.SpanContext)
}

// Empty check span is empty (无实质 trace 数据)
func (s *Span) Empty() bool {
	if s == nil || s.SpanContext == nil {
		return true
	}
	return s.Get("x-b3-traceid") == "" && s.Get("x-b3-spanid") == ""
}

// LogFields 记录日志字段到 span
func (s *Span) LogFields(fields ...log.Field) {
	if s != nil && s.Span != nil {
		s.Span.LogFields(fields...)
	}
}

// SetBaggage 设置 baggage item
func (s *Span) SetBaggage(key, value string) *Span {
	if s != nil && s.Span != nil {
		s.Span.SetBaggageItem(key, value)
	}
	return s
}

// Finish span
func (s *Span) Finish() {
	if s == nil || s.Span == nil {
		return
	}

	s.Span.Finish()
}
