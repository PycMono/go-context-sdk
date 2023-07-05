package bizctx

import (
	"context"
)

/***
 业务上下文
	1、将业务上下文和系统上下文分开，避免业务上下文污染系统上下文。
	2、业务上下文只专注业务字段透传，不关心系统上下文。
	3、系统上下文关注 reqid、rpc/http调用链等。
*/

const (
	// 租户id
	account string = "account"
)

// BizContext 业务上下文
type BizContext map[string]string

// Set 设置
func (c BizContext) Set(k, v string) BizContext {
	c[k] = v
	return c
}

// GetAccount 获取租户ID
func GetAccount(ctx context.Context) string {
	return getBizContext(ctx)[account]
}

// IsEmpty 是否为空
func (c BizContext) IsEmpty() bool {
	return len(c) == 0
}

// WithBizContext 设置ctx value并且返回ctx
func WithBizContext(ctx context.Context, in BizContext) context.Context {
	ctv := GetBizContext(ctx) // 先获取原值在进行追加防止被覆盖
	for k, v := range ctv {
		in[k] = v
	}

	return context.WithValue(ctx, bizCtxKey, in)
}

// GetBizContext get context
func GetBizContext(ctx context.Context) BizContext {
	origin := getBizContext(ctx)
	out := make(BizContext, len(origin))
	for k, v := range origin {
		out[k] = v
	}
	return out
}

func getBizContext(ctx context.Context) BizContext {
	bc := ctx.Value(bizCtxKey)
	if v, ok := bc.(BizContext); ok {
		return v
	}
	return make(BizContext)
}
