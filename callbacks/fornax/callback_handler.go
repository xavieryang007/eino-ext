package fornax

import (
	"context"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino-ext/callbacks/metrics"
)

// NewDefaultCallbackHandler customize with options
// Use fornax_sdk.NewClient to init first before NewDefaultCallbackHandler
func NewDefaultCallbackHandler(client *fornax_sdk.Client, opts ...Option) callbacks.Handler {
	var handlers []callbacks.Handler

	o := &options{
		enableTracing: true,
		enableMetrics: true,
	}

	for _, opt := range opts {
		opt(o)
	}

	if o.enableTracing {
		handlers = append(handlers, newTraceCallbackHandler(client, o))
	}

	if o.enableMetrics {
		handlers = append(handlers, newMetricsCallbackHandler(client, o))
		h, err := metrics.NewMetricsHandler()
		if err != nil {
			logs.Errorf("init metrics handler fail: %v", err)
		} else {
			handlers = append(handlers, h)
		}
	}

	return &fornaxTracer{handlers: handlers}
}

// Close should be called before service finished
func Close() {
	// close fornax trace
	fornax_sdk.Close()
}

type fornaxTracer struct {
	handlers []callbacks.Handler
}

func (f *fornaxTracer) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	for _, handler := range f.handlers {
		ctx = handler.OnStart(ctx, info, input)
	}

	return ctx
}

func (f *fornaxTracer) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	for _, handler := range f.handlers {
		ctx = handler.OnEnd(ctx, info, output)
	}

	return ctx
}

func (f *fornaxTracer) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	for _, handler := range f.handlers {
		ctx = handler.OnError(ctx, info, err)
	}

	return ctx
}

func (f *fornaxTracer) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if len(f.handlers) == 0 {
		input.Close()
		return ctx
	}

	inputs := input.Copy(len(f.handlers))

	for i := range inputs {
		ctx = f.handlers[i].OnStartWithStreamInput(ctx, info, inputs[i])
	}

	return ctx
}

func (f *fornaxTracer) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if len(f.handlers) == 0 {
		output.Close()
		return ctx
	}

	outputs := output.Copy(len(f.handlers))

	for i := range outputs {
		ctx = f.handlers[i].OnEndWithStreamOutput(ctx, info, outputs[i])
	}

	return ctx
}
