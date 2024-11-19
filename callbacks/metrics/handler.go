package metrics

import (
	"context"
	"io"
	"runtime/debug"
	"sync"
	"time"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/gopkg/metrics/v4"
)

var once sync.Once
var initErr error
var einoVersion string

func NewMetricsHandler() (callbacks.Handler, error) {
	once.Do(func() {
		einoVersion = getEinoVersion()
		initErr = initMetrics()
	})
	if initErr != nil {
		return nil, initErr
	}
	return &handler{}, nil
}

type startTimeKey struct{}

type handler struct{}

func (h *handler) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info == nil {
		return ctx
	}

	ctx = context.WithValue(ctx, startTimeKey{}, time.Now())
	return ctx
}

func (h *handler) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	emitEnd(ctx, info, false)
	return ctx
}

func (h *handler) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	emitEnd(ctx, info, true)
	return ctx
}

func (h *handler) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	input.Close()

	ctx = context.WithValue(ctx, startTimeKey{}, time.Now())
	return ctx
}

func (h *handler) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	startTime, ok := ctx.Value(startTimeKey{}).(time.Time)
	if ok {
		err := outputStreamStartMetric.WithTags(
			metrics.T{Name: tagNameRunInfoComponent, Value: string(info.Component)},
			metrics.T{Name: tagNameRunInfoType, Value: string(info.Type)},
			metrics.T{Name: tagNameRunInfoName, Value: string(info.Name)},
		).Emit(metrics.Observe(int(time.Now().Sub(startTime).Milliseconds())))
		if err != nil {
			logs.CtxError(ctx, "emit stream start metric fail: %s", err.Error())
		}
	}
	go func() {
		var err error
		defer func() {
			e := recover()
			if e != nil {
				logs.CtxError(ctx, "metrics OnEndWithStreamOutput callback panic: %v", e)
			}
			output.Close()
			emitEnd(ctx, info, (err != nil && err != io.EOF) || e != nil)
		}()

		for {
			_, err = output.Recv()
			if err == io.EOF {
				return
			}
			if err != nil {
				return
			}
		}
	}()
	return ctx
}

func emitEnd(ctx context.Context, info *callbacks.RunInfo, isError bool) {
	if info == nil {
		return
	}

	values := []*metrics.Value{
		metrics.Incr(1),
	}
	startTime, ok := ctx.Value(startTimeKey{}).(time.Time)
	if ok {
		values = append(values, metrics.Observe(int(time.Now().Sub(startTime).Milliseconds())))
	}

	isErrorStr := "0"
	if isError {
		isErrorStr = "1"
	}
	err := commonMetric.WithTags(
		metrics.T{Name: tagNameRunInfoComponent, Value: string(info.Component)},
		metrics.T{Name: tagNameRunInfoType, Value: string(info.Type)},
		metrics.T{Name: tagNameRunInfoName, Value: string(info.Name)},
		metrics.T{Name: tagNameIsError, Value: isErrorStr},
	).Emit(values...)
	if err != nil {
		logs.CtxError(ctx, "emit common metric fail: %s", err.Error())
	}

	if info.Component == compose.ComponentOfStateGraph ||
		info.Component == compose.ComponentOfGraph ||
		info.Component == compose.ComponentOfChain {
		err = graphMetric.WithTags(
			metrics.T{Name: tagNameSDKVersion, Value: einoVersion},
		).Emit(metrics.Incr(1))
		if err != nil {
			logs.CtxError(ctx, "emit psm metric fail: %s", err.Error())
		}
	}
	return
}

const einoImportPath = "code.byted.org/flow/eino"

func getEinoVersion() string {
	if v, ok := readVersionByGoMod(einoImportPath); ok {
		return v
	}

	return "unknown_build_info"
}

func readVersionByGoMod(path string) (string, bool) {
	buildInfo, ok := debug.ReadBuildInfo()
	if !ok {
		return "", false
	}

	for _, dep := range buildInfo.Deps {
		if dep.Path == path {
			if dep.Replace != nil {
				return dep.Replace.Version, true
			} else {
				return dep.Version, true
			}
		}
	}

	return "", false
}
