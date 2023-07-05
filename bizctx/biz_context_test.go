package bizctx

import (
	"context"
	"testing"
)

func TestContext(t *testing.T) {
	ctx := WithBizContext(context.TODO(), BizContext{})
	println(ctx)

	//ctx1 := New()
	//println(ctx1)
}
