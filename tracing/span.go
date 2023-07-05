package tracing

import (
	"context"
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
)

// Span is a wrapper of opentracing.Span
type Span struct {
	opentracing.Span
	SpanContext
}

// StartChildSpan start a child span
func (s *Span) StartChildSpan(operationName string, opts ...opentracing.StartSpanOption) Span {
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
	span.SetBaggageItem(TagRequestID, s.RequestID())

	// 设置traceID
	spanCtx, ok := span.Context().(jaeger.SpanContext)
	if !ok {
		return Span{Span: span}
	}
	span.SetBaggageItem("x-b3-traceid", spanCtx.TraceID().String())
	if req := span.BaggageItem(TagRequestID); len(req) == 0 {
		span.SetBaggageItem(TagRequestID, NewRequestID())
	}

	return Span{Span: span}
}

// SetTag set a tag to span
func (s *Span) SetTag(k string, v interface{}) *Span {
	s.Span.SetTag(k, v)
	return s
}

// WithContextSetOpentracing return a context with span
func (s *Span) WithContextSetOpentracing(ctx context.Context) context.Context {
	return opentracing.ContextWithSpan(ctx, s.Span)
}

// WithContext 设置context value
func (s *Span) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, activeSpanKey, s.SpanContext)
}

// Empty check span is empty
func (s *Span) Empty() bool {
	return s.SpanContext == nil || s.Span == nil
}

// Finish span
func (s *Span) Finish() {
	if s.Span == nil {
		return
	}

	s.Span.Finish()
}
