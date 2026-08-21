package tracing

import (
	"context"
	"strings"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// The GenAI and MCP attribute names in this file follow OpenTelemetry
// semantic-conventions revision 1.41.0. They are centralized here because
// these Development declarations were removed from the generic generated Go
// semconv package starting with v1.42.0.

// Field is one OpenTelemetry span attribute.
type Field = attribute.KeyValue

// KV creates a Field from an OpenTelemetry-native scalar or slice value.
// Blank keys and unsupported values return an invalid Field.
func KV(key string, value any) Field {
	if strings.TrimSpace(key) == "" {
		return Field{}
	}

	switch value := value.(type) {
	case string:
		return attribute.String(key, value)
	case bool:
		return attribute.Bool(key, value)
	case int:
		return attribute.Int(key, value)
	case int8:
		return attribute.Int64(key, int64(value))
	case int16:
		return attribute.Int64(key, int64(value))
	case int32:
		return attribute.Int64(key, int64(value))
	case int64:
		return attribute.Int64(key, value)
	case float32:
		return attribute.Float64(key, float64(value))
	case float64:
		return attribute.Float64(key, value)
	case []string:
		return attribute.StringSlice(key, value)
	case []bool:
		return attribute.BoolSlice(key, value)
	case []int:
		return attribute.IntSlice(key, value)
	case []int64:
		return attribute.Int64Slice(key, value)
	case []float64:
		return attribute.Float64Slice(key, value)
	default:
		return Field{}
	}
}

// WithKV adds fields to the current recording span and returns ctx unchanged.
// It does not create a span or retain fields for future spans.
func WithKV(ctx context.Context, fields ...Field) context.Context {
	span := trace.SpanFromContext(ctx)
	if !span.IsRecording() {
		return ctx
	}

	valid := make([]attribute.KeyValue, 0, len(fields))
	for _, field := range fields {
		if field.Valid() {
			valid = append(valid, field)
		}
	}
	if len(valid) != 0 {
		span.SetAttributes(valid...)
	}
	return ctx
}

// OperationName records the GenAI operation being performed.
func OperationName(value string) Field {
	return KV("gen_ai.operation.name", value)
}

// ProviderName records the GenAI provider name.
func ProviderName(value string) Field {
	return KV("gen_ai.provider.name", value)
}

// RequestModel records the model requested by the caller.
func RequestModel(value string) Field {
	return KV("gen_ai.request.model", value)
}

// ResponseModel records the model identifier returned by the provider.
func ResponseModel(value string) Field {
	return KV("gen_ai.response.model", value)
}

// FinishReasons records the normalized reasons a model response finished.
func FinishReasons(values ...string) Field {
	return KV("gen_ai.response.finish_reasons", values)
}

// InputTokens records total input token usage.
func InputTokens(value int) Field {
	return KV("gen_ai.usage.input_tokens", value)
}

// OutputTokens records total output token usage.
func OutputTokens(value int) Field {
	return KV("gen_ai.usage.output_tokens", value)
}

// ToolName records the name of a tool used by an agent.
func ToolName(value string) Field {
	return KV("gen_ai.tool.name", value)
}

// ToolCallID records the model-provided identifier of a tool call.
func ToolCallID(value string) Field {
	return KV("gen_ai.tool.call.id", value)
}

// ToolType records the semantic type of a tool.
func ToolType(value string) Field {
	return KV("gen_ai.tool.type", value)
}

// MCPMethodName records the MCP request or notification method.
func MCPMethodName(value string) Field {
	return KV("mcp.method.name", value)
}

// MCPProtocolVersion records the negotiated Model Context Protocol version.
func MCPProtocolVersion(value string) Field {
	return KV("mcp.protocol.version", value)
}

// MCPSessionID records the MCP session identifier.
func MCPSessionID(value string) Field {
	return KV("mcp.session.id", value)
}

// MCPResourceURI records the URI of an MCP resource operation.
func MCPResourceURI(value string) Field {
	return KV("mcp.resource.uri", value)
}
