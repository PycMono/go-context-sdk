# go-context-sdk

`go-context-sdk` 提供两类互不混用的 Context 能力：

- `tracing`：基于标准 OpenTelemetry API 的底层 Span 创建、W3C Trace Context 传播、TraceID 读取和当前 Span 属性补充。
- `bizctx`：进程内业务字段容器，包含 `id`、`userid`、`tenantid`、`appid`、`clientip`。

`trace_id` 是唯一技术关联 ID。本库不生成或传播独立 Request-ID，也不提供自定义 Span、私有 TracerProvider、Exporter 或兼容层。

## 安装

```bash
go get github.com/PycMono/go-context-sdk@v1.2.0
```

## 应用装配

应用在启动时创建并安装一次全局 OTel Provider 和 W3C Propagator。本库读取标准全局对象，不保存第二份状态：

```go
resource, err := resource.New(ctx,
	resource.WithAttributes(semconv.ServiceName("order-service")),
)
if err != nil {
	return err
}

tp := sdktrace.NewTracerProvider(
	sdktrace.WithResource(resource),
	sdktrace.WithBatcher(exporter),
)
otel.SetTracerProvider(tp)
otel.SetTextMapPropagator(propagation.TraceContext{})

defer tp.Shutdown(context.Background())
```

未安装 SDK Provider 时使用标准全局 Noop：不联网、不创建后台 goroutine、不记录 Span。合法入站父 TraceID 可以保留；无父调用不会生成兜底 ID。若关闭导出但仍需生成 TraceID，可安装无 Exporter、`NeverSample` 的 SDK Provider。

## 业务代码补充字段

Span 应由 HTTP Middleware、Provider Decorator、Tool Runtime、Repository Decorator、MQ/Job Runner 等框架层自动创建和结束。普通业务代码只传递 Context，需要补充排障维度时使用 `WithKV`：

```go
ctx = tracing.WithKV(ctx,
	tracing.KV("order.type", "standard"),
	tracing.KV("order.item_count", len(request.Items)),
)
```

`WithKV` 只更新调用时 Context 中的当前 Recording Span，不创建或结束 Span，也不会把字段保存在 Context 中等待未来 Span。无 Span、Noop Provider、空 key 或不支持的值类型均安全空转；返回值始终是原 Context。

## 模型、Tool 与 MCP 预设

`tracing/preset.go` 平级提供标准语义 Field，不需要引入额外子包：

```go
ctx = tracing.WithKV(ctx,
	tracing.OperationName("chat"),
	tracing.ProviderName("deepseek"),
	tracing.RequestModel("deepseek-chat"),
)

ctx = tracing.WithKV(ctx,
	tracing.ResponseModel(result.Model),
	tracing.FinishReasons(result.FinishReasons...),
	tracing.InputTokens(result.InputTokens),
	tracing.OutputTokens(result.OutputTokens),
)
```

Tool 预设为 `ToolName`、`ToolCallID`、`ToolType`；MCP 预设为 `MCPMethodName`、`MCPProtocolVersion`、`MCPSessionID`、`MCPResourceURI`。这些方法只生成 Field，不创建操作节点。Tool 参数、Tool 输出和描述正文没有预设，避免把敏感内容或大文本写入 Trace。

本库不提供 `Fail`。谁创建 Span，谁根据返回 error、结构化结果、取消或超时设置错误状态并结束 Span。

## 底层 instrumentation API

`StartSpan` 直接返回标准 `trace.Span`，用于跨 Module 的 Middleware、Decorator 和 Runner 实现：

```go
ctx, span := tracing.StartSpan(ctx, "create-order")
defer span.End()

ctx = tracing.WithKV(ctx, tracing.KV("order.type", "standard"))
return next(ctx)
```

普通业务接入不应直接使用 `StartSpan`。框架创建子 Span 后必须把返回的 Context 传给下游。

读取当前标准 Span 使用 `trace.SpanFromContext(ctx)`；读取用于日志或响应回执的唯一关联 ID 使用：

```go
traceID := tracing.TraceIDFromContext(ctx)
```

无有效 SpanContext 时返回空字符串。

## HTTP 传播

应用必须把全局 Propagator 设置为 `propagation.TraceContext{}`，不得加入 Baggage。

```go
ctx := tracing.Extract(request.Context(), request.Header)
ctx, span := tracing.StartSpan(ctx, "HTTP "+request.Method,
	trace.WithSpanKind(trace.SpanKindServer),
)
defer span.End()

request = request.WithContext(ctx)
if traceID := tracing.TraceIDFromContext(ctx); traceID != "" {
	response.Header().Set(tracing.HeaderTraceID, traceID)
}
```

`Extract` 对 nil、缺失、非法、多值或逗号合并的 `traceparent` 静默忽略；合法父上下文标记为 Remote。`Inject` 把当前 SpanContext 写为 W3C `traceparent`/`tracestate`：

```go
request, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
if err != nil {
	return err
}
tracing.Inject(ctx, request.Header)
```

生产 HTTP Client 推荐使用官方 instrumentation 自动创建 CLIENT Span 并传播 W3C：

```go
client := &http.Client{
	Transport: otelhttp.NewTransport(http.DefaultTransport),
}
```

公网入口不得直接信任外部 `traceparent`/`tracestate`。网关应终止外部上下文并创建内部 root Span；只有明确允许的可信上游可以续接 Remote Parent。TraceID 不得用于鉴权、幂等或业务身份。

## 非 HTTP 入口

- MQ Consumer：从消息 Carrier 提取 W3C Context，再创建 `SpanKindConsumer` Span；没有父上下文时创建 root Span。
- 定时任务、CLI：在第一条业务日志之前创建 root Span。
- 消息号、任务号、工单号等使用独立且有明确语义的业务字段，不得复用 TraceID。

## BizContext

```go
ctx = bizctx.WithKV(ctx,
	bizctx.ID("entity-1"),
	bizctx.UserID("user-1"),
	bizctx.TenantID("tenant-1"),
	bizctx.AppID("app-1"),
	bizctx.ClientIP("203.0.113.10"),
)

userID := bizctx.GetUserID(ctx)
```

BizContext 不包含 Request-ID。其 HTTP Header 映射由上层框架负责。

## v1.2.0 公开 tracing API

```go
func StartSpan(context.Context, string, ...trace.SpanStartOption) (context.Context, trace.Span)
type Field = attribute.KeyValue
func KV(string, any) Field
func WithKV(context.Context, ...Field) context.Context

func OperationName(string) Field
func ProviderName(string) Field
func RequestModel(string) Field
func ResponseModel(string) Field
func FinishReasons(...string) Field
func InputTokens(int) Field
func OutputTokens(int) Field
func ToolName(string) Field
func ToolCallID(string) Field
func ToolType(string) Field
func MCPMethodName(string) Field
func MCPProtocolVersion(string) Field
func MCPSessionID(string) Field
func MCPResourceURI(string) Field

func Extract(context.Context, http.Header) context.Context
func Inject(context.Context, http.Header)
func TraceIDFromContext(context.Context) string

const HeaderTraceID = "trace-id"
```

v1.2.0 直接发布终态 API，不包含旧 OpenTracing/Jaeger/B3、自定义 Span、`Init`、`ServiceName`、Request-ID、`Fail` 或 deprecated 适配接口。
