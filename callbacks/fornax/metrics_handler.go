package fornax

import (
	"context"
	"io"
	"strconv"
	"time"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/fornax_sdk"
	"code.byted.org/flowdevops/fornax_sdk/domain"
	"code.byted.org/flowdevops/fornax_sdk/infra/ob"
	"code.byted.org/gopkg/env"
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/obric/flow_telemetry_go/operational"
)

func newMetricsCallbackHandler(client *fornax_sdk.Client, o *options) callbacks.Handler {
	m := &einoMetrics{
		mtr:      ob.NewFornaxMetrics(client.CommonService.GetIdentity()),
		identity: client.CommonService.GetIdentity(),
	}

	return m
}

type einoMetrics struct {
	mtr      *ob.FornaxMetrics
	identity *domain.Identity
}

func (l *einoMetrics) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	if info != nil && isInfraComponent(info.Component) {
		ctx = setMetricsGraphName(ctx, info.Name)
	}

	return setMetricsVariablesValue(ctx, &metricsVariablesValue{
		startTime:     time.Now(),
		callbackInput: input,
	})
}

func (l *einoMetrics) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	if info == nil {
		return ctx
	}

	var (
		startTime time.Time

		status    = ob.TagValueStatusSuccess
		spaceID   = l.identity.GetSpaceID()
		graphName = getMetricsGraphName(ctx)
	)

	ctxVal, ctxValOK := getMetricsVariablesValue(ctx)
	if ctxValOK {
		startTime = ctxVal.startTime
	}

	switch info.Component {
	case components.ComponentOfChatModel:
		var (
			modelName string
			tokenInfo ob.TokenInfo
			ext       = model.ConvCallbackOutput(output)
		)

		if ext == nil {
			break
		} else if ext.Config != nil {
			modelName = ext.Config.Model
		}

		if ext.TokenUsage != nil {
			tokenInfo = ob.TokenInfo{
				PromptTokens:     ext.TokenUsage.PromptTokens,
				CompletionTokens: ext.TokenUsage.CompletionTokens,
				TotalTokens:      ext.TokenUsage.TotalTokens,
			}
		}

		qme := l.mtr.GetModelQueryMetricEmitter(ctx, status, graphName, modelName)
		qme.EmitMeter(1)
		qme.EmitLatency(ctx, startTime)

		l.mtr.GetModelFirstTokenMetricEmitter(ctx, graphName, modelName).EmitTokenLatency(ctx, startTime, tokenInfo)
		l.mtr.GetModelTokensMetricEmitter(ctx, graphName, modelName).EmitTokens(ctx, tokenInfo)

		ob.Report(ctx, operational.Params{
			GraphUID:     graphName,
			SpaceID:      strconv.FormatInt(spaceID, 10),
			IsBoe:        env.IsBoe(),
			ModelID:      modelName,
			Tokens:       int64(tokenInfo.TotalTokens),
			InputTokens:  int64(tokenInfo.PromptTokens),
			OutputTokens: int64(tokenInfo.CompletionTokens),
		}, l.getUserID(ctx))
	default:
		if isInfraComponent(info.Component) {
			queryEmitter := l.mtr.GetGraphQueryMetricEmitter(ctx, info.Name, status)
			queryEmitter.EmitMeter(1)
			queryEmitter.EmitLatency(ctx, startTime)

			l.mtr.GetGraphFirstTokenMetricEmitter(ctx, info.Name).EmitLatency(ctx, startTime)
		}
	}

	return ctx
}

func (l *einoMetrics) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	if info == nil {
		return ctx
	}

	var (
		startTime time.Time

		status    = ob.TagValueStatusError
		graphName = getMetricsGraphName(ctx)
	)

	ctxVal, ctxValOK := getMetricsVariablesValue(ctx)
	if ctxValOK {
		startTime = ctxVal.startTime
	}

	switch info.Component {
	case components.ComponentOfChatModel:
		var modelName string

		if ctxValOK {
			if input := model.ConvCallbackInput(ctxVal.callbackInput); input != nil && input.Config != nil {
				modelName = input.Config.Model
			}
		}

		qme := l.mtr.GetModelQueryMetricEmitter(ctx, status, graphName, modelName)
		qme.EmitMeter(1)
		qme.EmitLatency(ctx, startTime)

	default:
		if isInfraComponent(info.Component) {
			qme := l.mtr.GetGraphQueryMetricEmitter(ctx, graphName, status)
			qme.EmitMeter(1)
			qme.EmitLatency(ctx, startTime)

			l.mtr.GetGraphFirstTokenMetricEmitter(ctx, graphName).
				EmitLatency(ctx, startTime)
		}

	}

	return ctx
}

func (l *einoMetrics) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	// TODO: stream input parse
	input.Close()

	if info != nil && isInfraComponent(info.Component) {
		ctx = setMetricsGraphName(ctx, info.Name)
	}

	return setMetricsVariablesValue(ctx, &metricsVariablesValue{
		startTime: time.Now(),
	})
}

func (l *einoMetrics) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	go func() {
		defer func() {
			if e := recover(); e != nil {
				logs.Warnf("[einoMetrics][OnEndWithStreamOutput] recovered: %s", e)
			}

			output.Close()
		}()

		if info == nil {
			return
		}

		switch info.Component {
		case components.ComponentOfChatModel:
			if err := l.handleChatModelStreamOutput(ctx, info, output); err != nil {
				logs.Warnf("[einoMetrics][OnEndWithStreamOutput][ChatModel] process failed, err=%v", err)
			}

		default:
			if isInfraComponent(info.Component) {
				var startTime time.Time

				if ctxVal, ok := getMetricsVariablesValue(ctx); ok {
					startTime = ctxVal.startTime
				}

				qme := l.mtr.GetGraphQueryMetricEmitter(ctx, info.Name, ob.TagValueStatusError)
				qme.EmitMeter(1)
				qme.EmitLatency(ctx, startTime)

				l.mtr.GetGraphFirstTokenMetricEmitter(ctx, info.Name).
					EmitLatency(ctx, startTime)
			}
		}
	}()

	return ctx
}

func (l *einoMetrics) handleChatModelStreamOutput(ctx context.Context, _ *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) error {

	var (
		modelName string
		tokenInfo ob.TokenInfo
		startTime time.Time

		status    = ob.TagValueStatusSuccess
		graphName = getMetricsGraphName(ctx)
		spaceID   = l.identity.GetSpaceID()
	)

	ctxVal, ctxValOK := getMetricsVariablesValue(ctx)
	if ctxValOK {
		startTime = ctxVal.startTime

		if input := model.ConvCallbackInput(ctxVal.callbackInput); input != nil && input.Config != nil {
			modelName = input.Config.Model
		}
	}

	cnt := 0

	for {
		item, recvErr := output.Recv()
		if recvErr != nil {
			if recvErr != io.EOF {
				status = ob.TagValueStatusError
			}

			break
		}

		cbOutput := model.ConvCallbackOutput(item)
		if cbOutput != nil && cbOutput.TokenUsage != nil {
			tokenInfo.PromptTokens += cbOutput.TokenUsage.PromptTokens
			tokenInfo.CompletionTokens += cbOutput.TokenUsage.CompletionTokens
			tokenInfo.TotalTokens += cbOutput.TokenUsage.TotalTokens
		}

		if cnt == 0 {
			l.mtr.GetModelFirstTokenMetricEmitter(ctx, graphName, modelName).EmitTokenLatency(ctx, startTime, tokenInfo)
		}

		cnt++
	}

	qme := l.mtr.GetModelQueryMetricEmitter(ctx, status, graphName, modelName)
	qme.EmitMeter(1)
	qme.EmitLatency(ctx, startTime)

	// stream 失败不会有 token usage
	if status == ob.TagValueStatusSuccess {
		l.mtr.GetModelTokensMetricEmitter(ctx, graphName, modelName).EmitTokens(ctx, tokenInfo)
	}

	ob.Report(ctx, operational.Params{
		GraphUID:     graphName,
		SpaceID:      strconv.FormatInt(spaceID, 10),
		IsBoe:        env.IsBoe(),
		ModelID:      modelName,
		Tokens:       int64(tokenInfo.TotalTokens),
		InputTokens:  int64(tokenInfo.PromptTokens),
		OutputTokens: int64(tokenInfo.CompletionTokens),
	}, l.getUserID(ctx))

	return nil
}

func (l *einoMetrics) getUserID(ctx context.Context) string {
	if uid, ok := getUserID(ctx); ok {
		return uid
	}

	return "0"
}
