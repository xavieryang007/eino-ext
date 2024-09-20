package fornaxplugin

import (
	"github.com/cloudwego/kitex/client/callopt"

	"code.byted.org/flow/eino/components/tool"
)

type pluginOption struct {
	callOpts []callopt.Option
}

func WithKiteCallOptions(opts ...callopt.Option) tool.Option {
	return func(o *tool.Options) {
		if po, ok := o.ImplSpecificOption.(*pluginOption); ok {
			po.callOpts = append(po.callOpts, opts...)
		}
	}
}

func getPluginOption(opts ...tool.Option) *tool.Options {
	opt := &tool.Options{
		ImplSpecificOption: &pluginOption{
			callOpts: make([]callopt.Option, 0),
		},
	}

	for _, fn := range opts {
		fn(opt)
	}
	return opt
}
