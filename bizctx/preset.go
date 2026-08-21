package bizctx

import "context"

// KV 用于 Build 的键值对构建函数
type KV func(bc BizContext)

// 预设字段
func ID(v string) KV       { return func(bc BizContext) { bc[bizID] = v } }
func UserID(v string) KV   { return func(bc BizContext) { bc[bizUserID] = v } }
func TenantID(v string) KV { return func(bc BizContext) { bc[bizTenantID] = v } }
func AppID(v string) KV    { return func(bc BizContext) { bc[bizAppID] = v } }
func ClientIP(v string) KV { return func(bc BizContext) { bc[bizClientIP] = v } }

// GetID 从 context 获取实体 ID
func GetID(ctx context.Context) string {
	return getBizContext(ctx).Get(bizID)
}

// GetUserID 从 context 获取用户 ID
func GetUserID(ctx context.Context) string {
	return getBizContext(ctx).Get(bizUserID)
}

// GetTenantID 从 context 获取租户 ID
func GetTenantID(ctx context.Context) string {
	return getBizContext(ctx).Get(bizTenantID)
}

// GetAppID 从 context 获取应用 ID
func GetAppID(ctx context.Context) string {
	return getBizContext(ctx).Get(bizAppID)
}

// GetClientIP 从 context 获取客户端 IP
func GetClientIP(ctx context.Context) string {
	return getBizContext(ctx).Get(bizClientIP)
}

// WithKV 通过 KV 函数批量更新 context 中的 BizContext 字段。
// 只修改 KV 指定的字段，未涉及的字段保持不变。
func WithKV(ctx context.Context, kv ...KV) context.Context {
	bc := getBizContext(ctx)
	for _, fn := range kv {
		fn(bc)
	}

	return withBizContext(ctx, bc)
}
