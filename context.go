package ctxsdk

import (
	"context"
	"github.com/opentracing/opentracing-go"
)

// NewContext 获取带链路信息的context，一般是初始化使用
func NewContext() context.Context {
	ctx := context.Background()
	ctv := BizCtxValue{}
	ctv.SetTraceID()
	return context.WithValue(ctx, ctxKey, ctv)
}

// WithBizCtxValue 设置ctx value并且返回ctx
func WithBizCtxValue(ctx context.Context, in BizCtxValue) context.Context {
	ctv := GetBizCtxValue(ctx) // 先获取原值
	if ctv.IsEmpty() {
		// 尝试从span中获取链路信息
		if span := opentracing.SpanFromContext(ctx); span != nil {
			ctv = BizCtxValue4Span(span)
		}
	}

	// 设置数据防止被覆盖
	for k, v := range ctv {
		in[k] = v
	}

	return context.WithValue(ctx, ctxKey, in)
}

// GetBizCtxValue get context
func GetBizCtxValue(ctx context.Context) BizCtxValue {
	bc := ctx.Value(ctxKey)
	if v, ok := bc.(BizCtxValue); ok {
		return v
	}

	// 尝试从span获取
	if span := opentracing.SpanFromContext(ctx); span != nil {
		return BizCtxValue4Span(span)
	}
	return make(BizCtxValue)
}
