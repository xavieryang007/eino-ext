package fornax

type options struct {
	enableTracing bool
	enableMetrics bool
	parser        CallbackDataParser
}

type Option func(o *options)

func WithEnableTracing(enable bool) Option {
	return func(o *options) {
		o.enableTracing = enable
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
