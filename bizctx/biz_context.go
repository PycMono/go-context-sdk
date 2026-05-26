package bizctx

import "context"

/***
业务上下文
   1、将业务上下文和系统上下文分开，避免业务上下文污染系统上下文。
   2、业务上下文只专注业务字段透传，不关心系统上下文。
   3、系统上下文关注 reqid、rpc/http调用链等。
*/

const (
	bizID        = "id"
	bizUserID    = "userid"
	bizTenantID  = "tenantid"
	bizAppID     = "appid"
	bizClientIP  = "clientip"
	bizRequestID = "requestid"
)

// BizContext 业务上下文
type BizContext map[string]string

// Get 获取指定 key 的值
func (c BizContext) Get(k string) string {
	if c == nil {
		return ""
	}
	return c[k]
}

// Delete 删除指定 key
func (c BizContext) Delete(k string) {
	if c == nil {
		return
	}
	delete(c, k)
}

// Keys 返回所有 key 的列表
func (c BizContext) Keys() []string {
	if c == nil {
		return nil
	}
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	return keys
}

// IsEmpty 是否为空
func (c BizContext) IsEmpty() bool {
	return len(c) == 0
}

// withBizContext 将 in 合并到当前 context 的 BizContext 中并返回新的 context。
// in 中的字段优先级高于已有字段（新值覆盖旧值），已有字段中 in 未指定的字段会被保留。
func withBizContext(ctx context.Context, in BizContext) context.Context {
	if in == nil {
		in = make(BizContext)
	}
	ctv := getBizContext(ctx)

	out := make(BizContext, len(ctv)+len(in))
	for k, v := range ctv {
		out[k] = v
	}
	for k, v := range in {
		out[k] = v
	}

	return context.WithValue(ctx, activeBizCtxKey, out)
}

func getBizContext(ctx context.Context) BizContext {
	bc := ctx.Value(activeBizCtxKey)
	if v, ok := bc.(BizContext); ok && v != nil {
		out := make(BizContext, len(v))
		for k, val := range v {
			out[k] = val
		}
		return out
	}
	return make(BizContext)
}
