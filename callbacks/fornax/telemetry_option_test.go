package fornax

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/callbacks"
)

func TestWithInjectTraceSwitch(t *testing.T) {
	o := WithInjectTraceSwitch(func(ctx context.Context, runInfo *callbacks.RunInfo) (needInject bool) {
		return runInfo.Name == "hello"
	})

	opt := &options{}
	o(opt)

	assert.True(t, opt.injectTraceSwitch(context.Background(), &callbacks.RunInfo{Name: "hello"}))
}
