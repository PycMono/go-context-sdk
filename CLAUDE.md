# go-context-sdk Development Guide

## Architecture

The module has two independent packages:

- `tracing` is a thin facade over the standard OpenTelemetry global API.
- `bizctx` stores explicitly named business fields in a separate Context value.

The only technical correlation identifier is the OTel TraceID. Do not add a Request-ID, UUID fallback, B3 bridge, Baggage propagation, private Provider, or custom Span wrapper.

## Terminal tracing surface

```go
func StartSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span)
func WithSpan(ctx context.Context, name string, fn func(context.Context) error, opts ...WithSpanOption) error
type Field = attribute.KeyValue
func KV(key string, value any) Field
func WithKV(ctx context.Context, fields ...Field) context.Context
func Extract(ctx context.Context, h http.Header) context.Context
func Inject(ctx context.Context, h http.Header)
func TraceIDFromContext(ctx context.Context) string
const HeaderTraceID = "trace-id"
```

`StartSpan` obtains its tracer from `otel.Tracer` with scope name `github.com/PycMono/go-context-sdk` and version `v1.2.0`. It is a low-level instrumentation API for Middleware, Decorator, Provider, Tool Runtime, and Runner implementations; ordinary business examples must not expose `trace.Span` or manual `End` calls.

`WithSpan` owns the Span status and lifecycle for function-scoped work: fn only receives Context, errors are marked with a classified stable description (see `WithErrorClassifier`), and panics are marked then re-panicked unchanged. Streaming spans whose lifecycle spans multiple calls keep manual `StartSpan`/`End`.

`WithKV` updates only the current recording Span and returns the identical Context. It must not create a Span, retain fields for future spans, serialize unsupported values, or fail business execution. Keep all model, Tool, and MCP presets flat in `tracing/preset.go`; do not create preset subpackages. Do not add `Fail`: the layer that creates a Span owns status and lifecycle.

`Extract` and `Inject` call `otel.GetTextMapPropagator`; the application must install `propagation.TraceContext{}`. Library code must import OTel API only. OTel SDK packages are allowed in tests and examples.

The standard global Noop Provider must remain safe: no network, goroutines, local recording, or generated root TraceID. It may preserve a valid incoming parent TraceID as required by OTel semantics.

## W3C boundary

`Extract` accepts exactly one non-comma-joined `traceparent`, silently ignores invalid input, joins multiple `tracestate` field lines on a cloned Header, and leaves the caller's Header unchanged. Public ingress must terminate untrusted external Trace Context before internal propagation. TraceID is not an authorization, idempotency, or identity value.

## BizContext

Supported presets are `ID`, `UserID`, `TenantID`, `AppID`, and `ClientIP`. Keep copy-on-read/copy-on-write behavior. Do not add a generic correlation or request identifier without a separate business contract.

## Testing

- Install global Provider/Propagator state only inside tests and restore the previous values with `t.Cleanup`.
- Use the OTel SDK In-Memory SpanRecorder for Span behavior.
- Cover valid, missing, malformed, multi-value, comma-joined, and nil HTTP propagation inputs.
- Run `go test -race ./...`, `git diff --check`, confirm `go mod graph` contains no Jaeger/OpenTracing dependency, and confirm production Go files do not import `github.com/google/uuid`. The test/example OTel SDK may retain its own indirect UUID dependency.
