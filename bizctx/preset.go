package bizctx

import (
	"context"
)

// WithAccount 为上下文添加租户信息
func WithAccount(ctx context.Context, accountID string) context.Context {
	return WithBizContext(
		ctx, BizContext{
			account: accountID,
		},
	)
}
