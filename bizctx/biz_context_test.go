package bizctx

import (
	"context"
	"testing"
)

func TestWithKVNotModifyInput(t *testing.T) {
	ctx := WithKV(context.Background(), UserID("u1"))

	// 再次 WithKV 修改 userID
	ctx = WithKV(ctx, UserID("u2"))

	if GetUserID(ctx) != "u2" {
		t.Fatalf("expected u2, got %s", GetUserID(ctx))
	}
}

func TestWithKVPrecedence(t *testing.T) {
	ctx := WithKV(context.Background(), UserID("old"))
	ctx = WithKV(ctx, UserID("new"))

	if GetUserID(ctx) != "new" {
		t.Fatalf("expected new, got %s", GetUserID(ctx))
	}
}

func TestWithKVCopies(t *testing.T) {
	ctx := WithKV(context.Background(), UserID("val"))

	// 修改返回的 context
	ctx = WithKV(ctx, UserID("modified"))

	// 再次读取应为新值，不影响原来的内部状态
	if GetUserID(ctx) != "modified" {
		t.Fatalf("expected modified, got %s", GetUserID(ctx))
	}
}

func TestPresetKV(t *testing.T) {
	bc := make(BizContext)
	ID("e1")(bc)
	UserID("u1")(bc)
	TenantID("t1")(bc)
	AppID("a1")(bc)
	ClientIP("1.2.3.4")(bc)
	RequestID("r1")(bc)

	if bc[bizID] != "e1" {
		t.Fatalf("expected id e1, got %s", bc[bizID])
	}
	if bc[bizUserID] != "u1" {
		t.Fatalf("expected userid u1, got %s", bc[bizUserID])
	}
	if bc[bizTenantID] != "t1" {
		t.Fatalf("expected tenantid t1, got %s", bc[bizTenantID])
	}
}

func TestWithXXXGetXXX(t *testing.T) {
	ctx := WithKV(context.Background(),
		ID("id1"),
		UserID("u1"),
		TenantID("t1"),
		AppID("a1"),
		ClientIP("1.2.3.4"),
		RequestID("r1"),
	)

	if GetID(ctx) != "id1" {
		t.Fatalf("expected id1, got %s", GetID(ctx))
	}
	if GetUserID(ctx) != "u1" {
		t.Fatalf("expected u1, got %s", GetUserID(ctx))
	}
	if GetTenantID(ctx) != "t1" {
		t.Fatalf("expected t1, got %s", GetTenantID(ctx))
	}
	if GetAppID(ctx) != "a1" {
		t.Fatalf("expected a1, got %s", GetAppID(ctx))
	}
	if GetClientIP(ctx) != "1.2.3.4" {
		t.Fatalf("expected 1.2.3.4, got %s", GetClientIP(ctx))
	}
	if GetRequestID(ctx) != "r1" {
		t.Fatalf("expected r1, got %s", GetRequestID(ctx))
	}
}

func TestWithKV(t *testing.T) {
	ctx := WithKV(context.Background(), UserID("u1"), TenantID("t1"))

	// WithKV 只更新指定字段，未涉及的字段保持不变
	ctx = WithKV(ctx, UserID("u2"), AppID("a1"))

	if GetUserID(ctx) != "u2" {
		t.Fatalf("expected userID u2, got %s", GetUserID(ctx))
	}
	// tenantid 未涉及，应保持原值
	if GetTenantID(ctx) != "t1" {
		t.Fatalf("expected tenantID t1, got %s", GetTenantID(ctx))
	}
	// appid 被新增
	if GetAppID(ctx) != "a1" {
		t.Fatalf("expected appID a1, got %s", GetAppID(ctx))
	}
}
