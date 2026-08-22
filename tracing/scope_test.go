package tracing

import (
	"context"
	"errors"
	"testing"

	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

func TestWithSpanSuccessKeepsUnsetStatus(t *testing.T) {
	_, recorder := installRecordingProvider(t)

	var sawValidSpan bool
	err := WithSpan(context.Background(), "op", func(ctx context.Context) error {
		sawValidSpan = trace.SpanFromContext(ctx).SpanContext().IsValid()
		WithKV(ctx, KV("k", "v"))
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !sawValidSpan {
		t.Fatal("fn 必须看到含有效 Span 的 Context")
	}
	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended = %d, want 1", len(ended))
	}
	if ended[0].Status().Code != codes.Unset {
		t.Fatalf("status = %v, want Unset", ended[0].Status())
	}
	if got := ended[0].Attributes(); len(got) != 1 || string(got[0].Key) != "k" {
		t.Fatalf("attributes = %v, want fn 内 WithKV 生效", got)
	}
}

func TestWithSpanClassifiesErrorAndReturnsIt(t *testing.T) {
	_, recorder := installRecordingProvider(t)

	business := errors.New("db connection refused with password detail")
	err := WithSpan(context.Background(), "op", func(context.Context) error {
		return business
	}, WithErrorClassifier(func(error) string { return "contract_invalid" }))
	if !errors.Is(err, business) {
		t.Fatalf("error = %v, want 原样返回", err)
	}
	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended = %d, want 1", len(ended))
	}
	status := ended[0].Status()
	if status.Code != codes.Error || status.Description != "contract_invalid" {
		t.Fatalf("status = %v/%q", status.Code, status.Description)
	}
	if status.Description == business.Error() {
		t.Fatal("错误正文不得进入 Status 描述")
	}
}

func TestWithSpanDefaultErrorDescription(t *testing.T) {
	_, recorder := installRecordingProvider(t)
	if err := WithSpan(context.Background(), "op", func(context.Context) error {
		return errors.New("boom")
	}); err == nil {
		t.Fatal("want error")
	}
	if got := recorder.Ended()[0].Status().Description; got != "error" {
		t.Fatalf("description = %q, want error", got)
	}
}

func TestWithSpanPanicMarksErrorEndsAndRepanics(t *testing.T) {
	_, recorder := installRecordingProvider(t)

	defer func() {
		recovered := recover()
		if recovered != "boom" {
			t.Fatalf("recovered = %v, want 原样 re-panic", recovered)
		}
		ended := recorder.Ended()
		if len(ended) != 1 {
			t.Fatalf("panic 路径必须恰好结束一次，ended = %d", len(ended))
		}
		status := ended[0].Status()
		if status.Code != codes.Error || status.Description != "panic" {
			t.Fatalf("status = %v/%q", status.Code, status.Description)
		}
	}()
	_ = WithSpan(context.Background(), "op", func(context.Context) error {
		panic("boom")
	})
}

func TestWithSpanPreservesParentTraceID(t *testing.T) {
	provider, recorder := installRecordingProvider(t)
	parentCtx, parent := provider.Tracer("test-parent").Start(context.Background(), "parent")
	defer parent.End()

	if err := WithSpan(parentCtx, "child", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended = %d, want 1", len(ended))
	}
	if ended[0].Parent().TraceID() != parent.SpanContext().TraceID() {
		t.Fatal("child 必须续用父 TraceID")
	}
}

func TestWithSpanNoopProviderStillRunsBusiness(t *testing.T) {
	installNoopProvider(t)
	ran := false
	if err := WithSpan(context.Background(), "op", func(ctx context.Context) error {
		ran = trace.SpanFromContext(ctx) != nil
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Fatal("Noop 下 fn 必须照常执行")
	}
}

func TestWithSpanFinalSpanName(t *testing.T) {
	_, recorder := installRecordingProvider(t)

	late := ""
	err := WithSpan(context.Background(), "HTTP GET", func(context.Context) error {
		late = "GET /api/v1/items/:id"
		return nil
	}, WithFinalSpanName(func() string { return late }))
	if err != nil {
		t.Fatal(err)
	}
	if got := recorder.Ended()[0].Name(); got != "GET /api/v1/items/:id" {
		t.Fatalf("name = %q, want 路由模板名", got)
	}
}

func TestWithSpanFinalSpanNameEmptyKeepsOriginal(t *testing.T) {
	_, recorder := installRecordingProvider(t)
	if err := WithSpan(context.Background(), "original", func(context.Context) error {
		return nil
	}, WithFinalSpanName(func() string { return "" })); err != nil {
		t.Fatal(err)
	}
	if got := recorder.Ended()[0].Name(); got != "original" {
		t.Fatalf("name = %q, want original", got)
	}
}
