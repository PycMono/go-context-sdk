package ctxsdk

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	ctx := WithBizCtxValue(context.TODO(), BizCtxValue{})
	println(ctx)

	ctx = NewContext()
	println(ctx)
}
