package tracing

import (
	"context"
	"reflect"
	"testing"

	"go.opentelemetry.io/otel/attribute"
)

func TestKVSupportsDocumentedAttributeValues(t *testing.T) {
	tests := []struct {
		name  string
		value any
		want  attribute.KeyValue
	}{
		{name: "string", value: "value", want: attribute.String("key", "value")},
		{name: "bool", value: true, want: attribute.Bool("key", true)},
		{name: "int", value: 1, want: attribute.Int("key", 1)},
		{name: "int8", value: int8(2), want: attribute.Int64("key", 2)},
		{name: "int16", value: int16(3), want: attribute.Int64("key", 3)},
		{name: "int32", value: int32(4), want: attribute.Int64("key", 4)},
		{name: "int64", value: int64(5), want: attribute.Int64("key", 5)},
		{name: "float32", value: float32(1.5), want: attribute.Float64("key", 1.5)},
		{name: "float64", value: 2.5, want: attribute.Float64("key", 2.5)},
		{name: "strings", value: []string{"a", "b"}, want: attribute.StringSlice("key", []string{"a", "b"})},
		{name: "bools", value: []bool{true, false}, want: attribute.BoolSlice("key", []bool{true, false})},
		{name: "ints", value: []int{1, 2}, want: attribute.IntSlice("key", []int{1, 2})},
		{name: "int64s", value: []int64{3, 4}, want: attribute.Int64Slice("key", []int64{3, 4})},
		{name: "float64s", value: []float64{1.5, 2.5}, want: attribute.Float64Slice("key", []float64{1.5, 2.5})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := KV("key", tt.value); !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("KV() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestKVRejectsBlankKeysAndUnsupportedValues(t *testing.T) {
	for _, field := range []Field{
		KV("", "value"),
		KV("   ", "value"),
		KV("key", struct{ Secret string }{Secret: "private"}),
	} {
		if field.Valid() {
			t.Fatalf("field = %#v, want invalid", field)
		}
	}
}

func TestWithKVUpdatesOnlyCurrentRecordingSpan(t *testing.T) {
	_, recorder := installRecordingProvider(t)
	ctx, span := StartSpan(context.Background(), "operation")

	gotCtx := WithKV(ctx,
		KV("order.type", "premium"),
		KV("order.item_count", 2),
		Field{},
	)
	if gotCtx != ctx {
		t.Fatal("WithKV must return the original context")
	}
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended span count = %d, want 1", len(ended))
	}
	attributes := make(map[string]attribute.Value, len(ended[0].Attributes()))
	for _, field := range ended[0].Attributes() {
		attributes[string(field.Key)] = field.Value
	}
	if got := attributes["order.type"].AsString(); got != "premium" {
		t.Fatalf("order.type = %q", got)
	}
	if got := attributes["order.item_count"].AsInt64(); got != 2 {
		t.Fatalf("order.item_count = %d", got)
	}
}

func TestWithKVDoesNotRetainFieldsForFutureSpans(t *testing.T) {
	_, recorder := installRecordingProvider(t)
	base := context.Background()

	if got := WithKV(base, KV("order.type", "premium")); got != base {
		t.Fatal("WithKV must return the original context")
	}
	_, span := StartSpan(base, "later")
	span.End()

	ended := recorder.Ended()
	if len(ended) != 1 || len(ended[0].Attributes()) != 0 {
		t.Fatalf("later span attributes = %#v, want none", ended[0].Attributes())
	}
}

func TestWithKVIsNoopWithoutRecordingSpan(t *testing.T) {
	installNoopProvider(t)
	ctx := context.Background()
	if got := WithKV(ctx, KV("key", "value")); got != ctx {
		t.Fatal("WithKV must return the original context")
	}
}

func TestSemanticFieldPresets(t *testing.T) {
	tests := []struct {
		name string
		got  Field
		want attribute.KeyValue
	}{
		{name: "operation", got: OperationName("chat"), want: attribute.String("gen_ai.operation.name", "chat")},
		{name: "provider", got: ProviderName("deepseek"), want: attribute.String("gen_ai.provider.name", "deepseek")},
		{name: "request model", got: RequestModel("deepseek-chat"), want: attribute.String("gen_ai.request.model", "deepseek-chat")},
		{name: "response model", got: ResponseModel("deepseek-chat-v2"), want: attribute.String("gen_ai.response.model", "deepseek-chat-v2")},
		{name: "finish reasons", got: FinishReasons("stop", "tool_use"), want: attribute.StringSlice("gen_ai.response.finish_reasons", []string{"stop", "tool_use"})},
		{name: "input tokens", got: InputTokens(12), want: attribute.Int("gen_ai.usage.input_tokens", 12)},
		{name: "output tokens", got: OutputTokens(8), want: attribute.Int("gen_ai.usage.output_tokens", 8)},
		{name: "tool name", got: ToolName("web_search"), want: attribute.String("gen_ai.tool.name", "web_search")},
		{name: "tool call ID", got: ToolCallID("call-1"), want: attribute.String("gen_ai.tool.call.id", "call-1")},
		{name: "tool type", got: ToolType("extension"), want: attribute.String("gen_ai.tool.type", "extension")},
		{name: "MCP method", got: MCPMethodName("tools/call"), want: attribute.String("mcp.method.name", "tools/call")},
		{name: "MCP protocol", got: MCPProtocolVersion("2025-03-26"), want: attribute.String("mcp.protocol.version", "2025-03-26")},
		{name: "MCP session", got: MCPSessionID("session-1"), want: attribute.String("mcp.session.id", "session-1")},
		{name: "MCP resource", got: MCPResourceURI("file:///workspace/readme.md"), want: attribute.String("mcp.resource.uri", "file:///workspace/readme.md")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !reflect.DeepEqual(tt.got, tt.want) {
				t.Fatalf("preset = %#v, want %#v", tt.got, tt.want)
			}
		})
	}
}
