package fornax

import (
	"context"

	"code.byted.org/flow/eino/callbacks"
)

type injectTraceSwitchFn func(ctx context.Context, runInfo *callbacks.RunInfo) (needInject bool)

type options struct {
	enableTracing     bool
	injectTraceSwitch injectTraceSwitchFn
	enableMetrics     bool
	parser            CallbackDataParser
}

type Option func(o *options)

func WithEnableTracing(enable bool) Option {
	return func(o *options) {
		o.enableTracing = enable
	}
}

// WithInjectTraceSwitch inject traces into context under specific conditions.
// trace will be injected if function is not configured, but this may lead to performance issues.
func WithInjectTraceSwitch(fn injectTraceSwitchFn) Option {
	return func(o *options) {
		o.injectTraceSwitch = fn
	}
}

func WithEnableMetrics(enable bool) Option {
	return func(o *options) {
		o.enableMetrics = enable
	}
}

func WithCallbackDataParser(parser CallbackDataParser) Option {
	return func(o *options) {
		o.parser = parser
	}
}
