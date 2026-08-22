package tracing

import (
	"context"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ErrorClassifier maps a business error to a stable, low-cardinality status
// description (for example a project error code). It must never return error
// text, stacks, or other dynamic content.
type ErrorClassifier func(err error) string

// withSpanOptions holds WithSpan configuration.
type withSpanOptions struct {
	classify ErrorClassifier
	start    []trace.SpanStartOption
	name     func() string
}

// WithSpanOption customizes WithSpan.
type WithSpanOption func(*withSpanOptions)

// WithErrorClassifier sets the error-to-status-description classifier used
// when fn fails. Without it, failed spans use the fixed description "error".
func WithErrorClassifier(classify ErrorClassifier) WithSpanOption {
	return func(o *withSpanOptions) { o.classify = classify }
}

// WithStartOptions passes standard OTel span start options (span kind,
// attributes, links, timestamps) through to StartSpan.
func WithStartOptions(opts ...trace.SpanStartOption) WithSpanOption {
	return func(o *withSpanOptions) { o.start = append(o.start, opts...) }
}

// WithFinalSpanName 指定在 Span 结束前求值的最终名称，用于创建时名称未知
// 的场景（如 HTTP 路由模板要等 handler 执行后才能确定）。返回空串保持原名。
func WithFinalSpanName(name func() string) WithSpanOption {
	return func(o *withSpanOptions) { o.name = name }
}

// WithSpan creates a Span for the duration of fn and always ends it exactly
// once. fn receives only the derived Context and never touches trace.Span;
// the Span owns its own status and lifecycle:
//
//   - fn returns nil: the Span keeps the default Unset status.
//   - fn returns an error: Error status with the classified stable
//     description; the error is returned unchanged.
//   - fn panics: the Span is marked Error with the fixed description "panic"
//     and ended before the panic propagates unchanged.
//
// The global Noop provider remains safe: fn still runs and no recording,
// network, or goroutines are created.
func WithSpan(ctx context.Context, name string, fn func(context.Context) error, opts ...WithSpanOption) (err error) {
	var options withSpanOptions
	for _, opt := range opts {
		if opt != nil {
			opt(&options)
		}
	}

	ctx, span := StartSpan(ctx, name, options.start...)
	defer func() {
		if options.name != nil {
			if final := options.name(); final != "" {
				span.SetName(final)
			}
		}
		if recovered := recover(); recovered != nil {
			span.SetStatus(codes.Error, "panic")
			span.End()
			panic(recovered)
		}
		if err != nil {
			description := "error"
			if options.classify != nil {
				if classified := options.classify(err); classified != "" {
					description = classified
				}
			}
			span.SetStatus(codes.Error, description)
		}
		span.End()
	}()
	return fn(ctx)
}
