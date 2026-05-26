# go-context-sdk

基于 `context.Context` 的微服务上下文管理 SDK，将**系统链路追踪**与**业务上下文**分离，支持跨服务透传。

## 设计原则

1. **职责分离**：`tracing` 负责链路追踪（trace/span/requestID），`bizctx` 负责业务字段透传（userID/tenantID 等）。
2. **不污染系统上下文**：业务字段通过独立的 key 存入 context，避免与 Go 标准库冲突。
3. **跨服务传播**：tracing 通过标准 B3 HTTP Header 透传；bizctx 的 HTTP 传输协议由上层框架 SDK（如 `go-gin-sdk`）实现。
4. **白名单收敛**：BizContext 的字段命名受预设 API 管控，防止 key 命名混乱。
5. **框架无关**：不提供任何 Web 框架（gin/echo/fasthttp）的专用中间件，框架适配由上层 SDK 或业务方自行实现。

---

## 目录

- [tracing 链路追踪](#tracing-链路追踪)
  - [创建 Span](#创建-span)
  - [从 Context 启动 Span](#从-context-启动-span)
  - [子 Span](#子-span)
  - [日志与 Baggage](#日志与-baggage)
  - [HTTP 注入与提取](#http-注入与提取)
- [bizctx 业务上下文](#bizctx-业务上下文)
  - [预设字段与便捷 API](#预设字段与便捷-api)
  - [KV 批量更新](#kv-批量更新)
- [在 Gin 中使用](#在-gin-中使用)
- [注意事项](#注意事项)

---

## tracing 链路追踪

### 创建 Span

```go
span := tracing.StartNewSpan("operation-name")
defer span.Finish()
```

### 从 Context 启动 Span

如果 context 里已有上游传入的 trace 信息，会自动恢复为 parent span：

```go
span := tracing.StartSpanFromContext(ctx, "operation-name")
defer span.Finish()
```

### 子 Span

```go
parent := tracing.StartNewSpan("parent-op")
defer parent.Finish()

child := parent.StartChildSpan("child-op")
child.SetTag("key", "value")
child.Finish()
```

### 日志与 Baggage

```go
import "github.com/opentracing/opentracing-go/log"

span.LogFields(
    log.String("event", "order-created"),
    log.Int("order_id", 42),
)
span.SetBaggage("tenant", "t1")
```

### HTTP 注入与提取

tracing 提供标准 B3 协议的 HTTP 注入与提取，适用于任何 HTTP client/server：

**Client 侧** —— 将 tracing 注入到下游请求 Header：

```go
req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
tracing.Inject(req, span)
http.DefaultClient.Do(req)
```

**Server 侧** —— 从请求 Header 提取：

```go
span, err := tracing.Extract(request)
defer span.Finish()
ctx := span.WithOpentracingContext(request.Context())
```

---

## bizctx 业务上下文

### 预设字段与便捷 API

BizContext 提供**白名单收敛**的预设字段，通过 `WithKV` 设置，通过 `GetXXX` 读取：

| 预设 KV | key | getter | 说明 |
|---------|-----|--------|------|
| `ID(v)` | `id` | `GetID` | 实体 ID |
| `UserID(v)` | `userid` | `GetUserID` | 用户/员工 ID |
| `TenantID(v)` | `tenantid` | `GetTenantID` | 租户/组织 ID |
| `AppID(v)` | `appid` | `GetAppID` | 应用/客户端 ID |
| `ClientIP(v)` | `clientip` | `GetClientIP` | 来源 IP |
| `RequestID(v)` | `requestid` | `GetRequestID` | 请求 ID |

**用法示例：**

```go
import "github.com/PycMono/go-context-sdk/bizctx"

// 设置单个/多个字段（未涉及的字段保持不变）
ctx = bizctx.WithKV(ctx, bizctx.UserID("u1"), bizctx.TenantID("t1"))

// 获取单个字段
userID := bizctx.GetUserID(ctx)
tenantID := bizctx.GetTenantID(ctx)

```

### KV 批量更新

通过 `WithKV` 批量更新 context 中的 BizContext 字段，**未涉及的字段保持不变**：

```go
// 设置多个字段
ctx = bizctx.WithKV(ctx,
    bizctx.UserID("u1"),
    bizctx.TenantID("t1"),
    bizctx.AppID("app1"),
    bizctx.ClientIP("1.2.3.4"),
    bizctx.RequestID("req-001"),
    bizctx.ID("entity-001"),
)

// 只更新 userID 和 appID，tenantID 保持 "t1" 不变
ctx = bizctx.WithKV(ctx,
    bizctx.UserID("u2"),
    bizctx.AppID("app1"),
)
```

### 跨服务传播

**bizctx 不提供 HTTP 传输协议**。bizctx 的跨服务透传（Header 编解码、白名单过滤）由上层框架 SDK（如 `go-gin-sdk`）实现。

**内存中使用：**

```go
// 从 context 读取
userID := bizctx.GetUserID(ctx)

// 写入 context
ctx = bizctx.WithUserID(ctx, "u1")
```

**HTTP 场景（由上层框架处理）：**

```go
// 在 go-gin-sdk 的中间件中统一实现
// 上游请求 header: x-bizctx-userid: u1
// 提取后: ctx = bizctx.WithKV(ctx, bizctx.UserID("u1"))
```

---

## 在 Gin 中使用

本 SDK **不提供 gin 专用中间件**，建议由上层 `go-gin-sdk` 或业务方自行封装。以下是参考实现：

### bizctx Gin 中间件（由 go-gin-sdk 实现）

```go
import "github.com/gin-gonic/gin"
import "github.com/PycMono/go-context-sdk/bizctx"

func BizctxMiddleware(c *gin.Context) {
    // 从 HTTP Header 提取 bizctx 字段（需自行实现 header 编解码 + 白名单过滤）
    ctx := c.Request.Context()
    if v := c.GetHeader("x-bizctx-userid"); v != "" {
        ctx = bizctx.WithKV(ctx, bizctx.UserID(v))
    }
    if v := c.GetHeader("x-bizctx-tenantid"); v != "" {
        ctx = bizctx.WithKV(ctx, bizctx.TenantID(v))
    }
    // ... 其他字段

    c.Request = c.Request.WithContext(ctx)
    c.Next()
}
```

### tracing Gin 中间件（由 go-gin-sdk 实现）

```go
import "github.com/PycMono/go-context-sdk/tracing"

func TracingMiddleware(c *gin.Context) {
    span, _ := tracing.Extract(c.Request)
    defer span.Finish() // ⚠️ 必须 finish，否则 trace 数据不落盘

    ctx := span.WithOpentracingContext(c.Request.Context())
    c.Request = c.Request.WithContext(ctx)

    c.Next()

    // gin 的 Next() 之后可补充 response tag
    span.SetTag("http.status_code", c.Writer.Status())
}
```

### 使用方式

```go
r := gin.Default()
r.Use(BizctxMiddleware, TracingMiddleware)

r.GET("/order", func(c *gin.Context) {
    span := tracing.StartSpanFromContext(c.Request.Context(), "order-handler")
    defer span.Finish()

    userID := bizctx.GetUserID(c.Request.Context())
    // ...
})
```

---

## 注意事项

1. **BizContext 是 string-string map**：所有值必须是字符串。需要存储复杂类型时，建议序列化为 JSON 字符串。
2. **WithKV 返回新 context**：符合 context 不可变语义，旧 context 不受影响。
4. **不提供任意 key 的 WithValue/Get**：为避免 key 命名混乱和 header 注入风险，所有业务字段必须通过预设的 `WithXXX`/`GetXXX` API 访问。
5. **框架无关**：不提供 gin/echo/fasthttp 专用中间件，bizctx 的 HTTP 传输由上层 SDK 或业务方自行实现。
6. **tracing 的 B3 协议是标准**：tracing 的 `Inject`/`Extract` 直接可用，无需额外封装。
7. **Jaeger 初始化**：`tracing` 包在 `init()` 中自动从环境变量初始化 Jaeger tracer。默认需要以下环境变量：
   - `JAEGER_SERVICE_NAME`：服务名
   - `JAEGER_AGENT_HOST` / `JAEGER_AGENT_PORT`：或 `JAEGER_COLLECTOR_ENDPOINT`
   - `JAEGER_SAMPLER_TYPE` / `JAEGER_SAMPLER_PARAM`：采样策略

---

## License

MIT
