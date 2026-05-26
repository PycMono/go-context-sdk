# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A Go library providing two separate context propagation systems built on top of Go's `context.Context`:

- **`bizctx`** — Business context propagation (userID, tenantID, etc.) independent of system context
- **`tracing`** — Distributed tracing wrapper around OpenTracing/Jaeger with B3 header propagation

## Build & Test

```bash
# Build all packages
go build ./...

# Run all tests
go test ./...

# Run a single test
go test ./bizctx -run TestSetNilMap

# Run example HTTP server (listens on :5005)
go run ./example
```

## Architecture

### bizctx — Business Context

`BizContext` is a `map[string]string` that stores business fields (userID, tenantID, appID, etc.) separately from Go's `context.Context`.

Key design points:
- Uses an unexported `contextKey` struct (`activeBizCtxKey`) to store the map in `context.Context` via `context.WithValue`
- **Always returns copies**: `GetBizContext`, `WithBizContext`, and `getBizContext` all return defensive copies so mutations don't leak across context boundaries
- `WithBizContext(ctx, in)` merges `in` on top of existing values (new values win, old values are preserved)
- `WithValue` sets a single key on the existing BizContext and returns a new context
- Preset helpers in `preset.go` (e.g. `UserID("u1")`) are `KV` functions that populate a `BizContext` map

HTTP propagation: `Inject`/`Extract` use headers with `x-bizctx-` prefix (lowercase keys in headers).

### tracing — Distributed Tracing

Wraps OpenTracing/Jaeger with a `Span` struct that combines an `opentracing.Span` with a `SpanContext` (a `map[string]string`).

Key design points:
- **Auto-initializes on import**: `tracing.go` has an `init()` that reads Jaeger config from env vars and starts a goroutine. The tracer is set as the global OpenTracing tracer
- `SpanContext` stores B3-style headers (`x-b3-traceid`, `x-b3-spanid`, `x-b3-sampled`) plus custom fields (`request-id`, `x-auth-accountid`, etc.)
- `SpanFromContext` looks up the span first from our custom context key, then falls back to `opentracing.SpanFromContext`
- `StartChildSpan` carries over `AccountID` and `RequestID` baggage from parent to child spans
- `SpanContext.SwapOpentracingSpanContext()` converts our map representation to/from Jaeger's `SpanContext` type for interop with the Jaeger client
- Request IDs are generated via `NewRequestID()` using UUID v4 with a fixed digit modification (char 15 replaced with `9`)

HTTP propagation: `Inject`/`Extract` use standard B3 headers plus custom headers (`request-id`, `x-auth-accountid`, etc.). `Extract` has an allowlist of known headers; additional headers can be passed via `extHeaders`.

### Cross-service flow (example)

The `example/http/server.go` demonstrates the typical pattern:
1. Incoming request arrives — extract tracing span and bizctx from headers
2. Start a child span for the operation
3. Merge bizctx into the Go context
4. When making downstream HTTP calls: `tracing.Inject(req, span)` and `bizctx.Inject(req, bc)` to propagate both contexts via headers

## Environment Variables (tracing)

The Jaeger tracer reads standard env vars at init time:
- `JAEGER_SERVICE_NAME` / `SERVICE_NAME` — service name
- `JAEGER_AGENT_HOST`, `JAEGER_AGENT_PORT` — agent endpoint
- `JAEGER_COLLECTOR_ENDPOINT` — collector endpoint
- `JAEGER_SAMPLER_TYPE`, `JAEGER_SAMPLER_PARAM` — sampling config

If no sampler config is provided, defaults to `const` sampler with param `1` (sample everything).
