package tracing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/uber/jaeger-client-go"
	"strconv"
)

// SpanContext is a wrapper of opentracing.SpanContext
type SpanContext map[string]string

// Set a key-value to SpanContext
func (s SpanContext) Set(key, value string) {
	s[key] = value
}

// Get a value from SpanContext
func (s SpanContext) Get(key string) string {
	return s[key]
}

// AccountID get account id from span context
func (s SpanContext) AccountID() string {
	return s.Get(HeaderAuthAccountID)
}

// SpanID get span id from span context
func (s SpanContext) SpanID() string {
	return s.Get("x-b3-spanid")
}

// RequestID get request id from span context
func (s SpanContext) RequestID() string {
	return s.Get("x-b3-traceid")
}

// SwapOpentracingSpanContext swap SpanContext to opentracing.SpanContext
func (s SpanContext) SwapOpentracingSpanContext() (opentracing.SpanContext, error) {
	var (
		traceID  jaeger.TraceID
		spanID   uint64
		parentID uint64
		sampled  = false
		baggage  = make(map[string]string)
		err      error
	)

	for key, value := range s {
		if key == "x-b3-traceid" {
			traceID, err = jaeger.TraceIDFromString(value)
		} else if key == "x-b3-parentspanid" {
			parentID, err = strconv.ParseUint(value, 16, 64)
		} else if key == "x-b3-spanid" {
			spanID, err = strconv.ParseUint(value, 16, 64)
		} else if key == "x-b3-sampled" && (value == "1" || value == "true") {
			sampled = true
		} else {
			baggage[key] = value
		}
		if err != nil {
			return nil, err
		}
	}

	return jaeger.NewSpanContext(
		traceID,
		jaeger.SpanID(spanID),
		jaeger.SpanID(parentID),
		sampled,
		baggage), nil
}
