package fornaxplugin

import (
	"github.com/cloudwego/kitex/client/callopt"

	"code.byted.org/flow/eino/components/tool"
)

type pluginOptions struct {
	callOpts []callopt.Option
}

func WithKiteCallOptions(opts ...callopt.Option) tool.Option {
	return tool.WrapImplSpecificOptFn(func(o *pluginOptions) {
		o.callOpts = append(o.callOpts, opts...)
	})
}
