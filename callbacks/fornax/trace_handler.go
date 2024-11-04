package fornax

import (
	"context"
	"time"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/flowdevops/fornax_sdk/infra/ob"
	"code.byted.org/gopkg/logs/v2"
)

func newTraceCallbackHandler(client *fornax_sdk.Client, o *options) callbacks.Handler {
	tracer := &einoTracer{
		client: client,
		parser: &defaultDataParser{},
	}

	if o.parser != nil {
		tracer.parser = o.parser
	}

	return tracer
}

type einoTracer struct {
	client *fornax_sdk.Client
	parser CallbackDataParser
}

func (l *einoTracer) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil {
		return ctx
	}

	spanName := info.Name
	if spanName == "" {
		spanName = string(info.Component)
	}

	span, ctx, err := l.client.StartSpan(
		ctx,
		spanName,
		parseSpanTypeFromComponent(info.Component),
		ob.AsyncChildSpan())
	if err != nil {
		logs.Warnf("[einoTracer][OnStart] start span failed: %s", err.Error())
		return ctx
	}

	si, ok := span.(*ob.FornaxSpanImpl)
	if !ok {
		logs.Warnf("[einoTracer][OnStart] span type assertion failed, actual=%T", si)
		return ctx
	}

	l.setRunInfo(ctx, si, info)

	if l.parser != nil {
		si.SetTag(ctx, l.parser.ParseInput(ctx, info, input))
	}

	return setTraceVariablesValue(ctx, &traceVariablesValue{
		startTime: time.Now(),
	})
}

func (l *einoTracer) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info == nil {
		return ctx
	}

	span := l.client.GetSpanFromContext(ctx)
	if span == nil {
		logs.Warn("[einoTracer][OnEnd] span not found in callback ctx")
		return ctx
	}

	si, ok := span.(*ob.FornaxSpanImpl)
	if !ok {
		logs.Warnf("[einoTracer][OnEnd] span type assertion failed, actual=%T", si)
		return ctx
	}

	if l.parser != nil {
		si.SetTag(ctx, l.parser.ParseOutput(ctx, info, output))
	}

	if stopCh, ok := ctx.Value(traceStreamInputAsyncKey{}).(streamInputAsyncVal); ok {
		<-stopCh
	}

	span.Finish(ctx)

	return ctx
}

func (l *einoTracer) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info == nil {
		return ctx
	}

	span := l.client.GetSpanFromContext(ctx)
	if span == nil {
		logs.Warn("[einoTracer][OnError] span not found in callback ctx")
		return ctx
	}

	si, ok := span.(*ob.FornaxSpanImpl)
	if !ok {
		logs.Warnf("[einoTracer][OnError] span type assertion failed, actual=%T", si)
		return ctx
	}

	si.SetTag(ctx, getErrorTags(ctx, err))

	if stopCh, ok := ctx.Value(traceStreamInputAsyncKey{}).(streamInputAsyncVal); ok {
		<-stopCh
	}

	span.Finish(ctx)

	return ctx
}

func (l *einoTracer) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	if info == nil {
		return ctx
	}

	spanName := info.Name
	if spanName == "" {
		spanName = string(info.Component)
	}

	span, ctx, err := l.client.StartSpan(ctx,
		spanName,
		parseSpanTypeFromComponent(info.Component),
		ob.SetStartTime(time.Now()),
		ob.AsyncChildSpan())
	if err != nil {
		logs.Warnf("[einoTracer][OnStartWithStreamInput] start span failed: %s", err.Error())
		return ctx
	}

	stopCh := make(streamInputAsyncVal)
	ctx = context.WithValue(ctx, traceStreamInputAsyncKey{}, stopCh)

	si, ok := span.(*ob.FornaxSpanImpl)
	if !ok {
		logs.Warnf("[einoTracer][OnStartWithStreamInput] span type assertion failed, actual=%T", si)
		return ctx
	}

	l.setRunInfo(ctx, si, info)

	if l.parser != nil {
		go func() {
			defer func() {
				if e := recover(); e != nil {
					logs.Warnf("[einoTracer][OnStartWithStreamInput] recovered: %s", e)
				}

				close(stopCh)
			}()

			si.SetTag(ctx, l.parser.ParseStreamInput(ctx, info, input))
		}()
	}

	return setTraceVariablesValue(ctx, &traceVariablesValue{
		startTime: time.Now(),
	})
}

func (l *einoTracer) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	if info == nil {
		return ctx
	}

	span := l.client.GetSpanFromContext(ctx)
	if span == nil {
		logs.Warn("[einoTracer][OnEndWithStreamOutput] span not found in callback ctx")
		return ctx
	}

	si, ok := span.(*ob.FornaxSpanImpl)
	if !ok {
		logs.Warnf("[einoTracer][OnEndWithStreamOutput] span type assertion failed, actual=%T", si)
		return ctx
	}

	if l.parser != nil {
		go func() {
			defer func() {
				if e := recover(); e != nil {
					logs.Warnf("[einoTracer][OnEndWithStreamOutput] recovered: %s", e)
				}
			}()

			si.SetTag(ctx, l.parser.ParseStreamOutput(ctx, info, output))

			if stopCh, ok := ctx.Value(traceStreamInputAsyncKey{}).(streamInputAsyncVal); ok {
				<-stopCh
			}

			span.Finish(ctx)
		}()
	}

	return ctx
}

func (l *einoTracer) setRunInfo(ctx context.Context, span *ob.FornaxSpanImpl, info *callbacks.RunInfo) {
	span.SetTag(ctx, make(spanTags).
		set(customSpanTagKeyComponent, string(info.Component)).
		set(customSpanTagKeyName, info.Name).
		set(customSpanTagKeyType, info.Type),
	)
	span.SetLibrary(ctx, einoLibrary)
	span.SetEinoVersion(ctx, getEinoSdkVersion())
}

// setFornaxTags set fornax inner tags
// Deprecated, because fornax span do the th
func (l *einoTracer) setFornaxTags(ctx context.Context, span *ob.FornaxSpanImpl) {
	//tags := make(spanTags).
	//	set("psm_env", env.Env()).
	//	set(obtag.SpaceID, itoa(l.identity.GetSpaceID())).
	//	set(obtag.FornaxSpaceID, itoa(l.identity.GetSpaceID())).
	//  set(obtag.Runtime, toJson(getStaticRuntimeTags(), false))
	//
	//if uid, ok := getUserID(ctx); ok {
	//	tags.set(obtag.UserID, uid)
	//}
	//
	//if did, ok := getDeviceID(ctx); ok {
	//	tags.set(obtag.DeviceID, did)
	//}
	//
	//if tid, ok := getThreadID(ctx); ok {
	//	tags.set(obtag.ThreadID, tid)
	//}
	//
	//span.SetTag(ctx, tags)
}
