package fornax

import (
	"context"
	"io"
	"time"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/indexer"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/components/prompt"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/flow-telemetry-common/go/obtag"
)

// CallbackDataParser tag parser for trace
// Implement CallbackDataParser and replace defaultDataParser by WithCallbackDataParser if needed
type CallbackDataParser interface {
	ParseInput(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) map[string]any
	ParseOutput(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) map[string]any
	ParseStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) map[string]any
	ParseStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) map[string]any
}

func NewDefaultDataParser() CallbackDataParser {
	return &defaultDataParser{}
}

type defaultDataParser struct{}

func (d defaultDataParser) ParseInput(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) map[string]any {
	if info == nil {
		return nil
	}

	tags := make(spanTags)

	switch info.Component {
	case components.ComponentOfChatModel:
		cbInput := model.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, convertModelInput(cbInput))

			if cbInput.Config != nil {
				tags.set(obtag.ModelName, cbInput.Config.Model)
				tags.set(obtag.CallOptions, convertModelCallOption(cbInput.Config))
			}
		}

		tags.set(obtag.ModelProvider, info.Type)

	case components.ComponentOfPrompt:
		cbInput := prompt.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, convertPromptInput(cbInput))
			tags.setFromExtraIfNotZero(obtag.PromptKey, cbInput.Extra)
			tags.setFromExtraIfNotZero(obtag.PromptVersion, cbInput.Extra)
			tags.setFromExtraIfNotZero(obtag.PromptProvider, cbInput.Extra)
		}

	case components.ComponentOfEmbedding:
		cbInput := embedding.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, cbInput.Texts)

			if cbInput.Config != nil {
				tags.set(obtag.ModelName, cbInput.Config.Model)
			}
		}

	case components.ComponentOfRetriever:
		cbInput := retriever.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, parseAny(ctx, cbInput.Query, false))
			tags.set(obtag.CallOptions, convertRetrieverCallOption(cbInput))

			tags.setFromExtraIfNotZero(obtag.VikingDBName, cbInput.Extra)
			tags.setFromExtraIfNotZero(obtag.VikingDBRegion, cbInput.Extra)

			tags.setFromExtraIfNotZero(obtag.ESName, cbInput.Extra)
			tags.setFromExtraIfNotZero(obtag.ESIndex, cbInput.Extra)
			tags.setFromExtraIfNotZero(obtag.ESCluster, cbInput.Extra)
		}

		tags.set(obtag.RetrieverProvider, info.Type)

	case components.ComponentOfIndexer:
		cbInput := indexer.ConvCallbackInput(input)
		if cbInput != nil {
			// rewrite if not suitable here
			tags.set(obtag.Input, parseAny(ctx, cbInput.Docs, false))
		}

	case compose.ComponentOfLambda:
		tags.set(obtag.Input, parseAny(ctx, input, false))

	default:
		tags.set(obtag.Input, parseAny(ctx, input, false))
	}

	return tags
}

func (d defaultDataParser) ParseOutput(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) map[string]any {
	if info == nil {
		return nil
	}

	tags := make(spanTags)

	switch info.Component {
	case components.ComponentOfChatModel:
		cbOutput := model.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, convertModelOutput(cbOutput))

			if cbOutput.TokenUsage != nil {
				tags.set(obtag.Tokens, cbOutput.TokenUsage.TotalTokens).
					set(obtag.InputTokens, cbOutput.TokenUsage.PromptTokens).
					set(obtag.OutputTokens, cbOutput.TokenUsage.CompletionTokens)
			}
		}

		tags.set(obtag.Stream, false)

		if tv, ok := getTraceVariablesValue(ctx); ok {
			tags.set(obtag.LatencyFirstResp, time.Since(tv.startTime).Milliseconds())
		}

	case components.ComponentOfPrompt:
		cbOutput := prompt.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, convertPromptOutput(cbOutput))
		}

	case components.ComponentOfEmbedding:
		cbOutput := embedding.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, parseAny(ctx, cbOutput.Embeddings, false))

			if cbOutput.TokenUsage != nil {
				tags.set(obtag.Tokens, cbOutput.TokenUsage.TotalTokens).
					set(obtag.InputTokens, cbOutput.TokenUsage.PromptTokens).
					set(obtag.OutputTokens, cbOutput.TokenUsage.CompletionTokens)
			}

			if cbOutput.Config != nil {
				tags.set(obtag.ModelName, cbOutput.Config.Model)
			}
		}

	case components.ComponentOfIndexer:
		cbOutput := indexer.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, parseAny(ctx, cbOutput.IDs, false))
		}

	case components.ComponentOfRetriever:
		cbOutput := retriever.ConvCallbackOutput(output)
		if cbOutput != nil {
			// rewrite if not suitable here
			tags.set(obtag.Output, convertRetrieverOutput(cbOutput))
		}

	case compose.ComponentOfLambda:
		tags.set(obtag.Output, parseAny(ctx, output, false))

	default:
		tags.set(obtag.Output, parseAny(ctx, output, false))

	}

	return tags
}

func (d defaultDataParser) ParseStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) map[string]any {
	defer input.Close()

	if info == nil {
		return nil
	}

	tags := make(spanTags)

	switch info.Component {
	default:
		chunks, recvErr := d.ParseDefaultStreamInput(ctx, input)
		if recvErr != nil {
			return tags.setTags(getErrorTags(ctx, recvErr))
		}

		tags.set(obtag.Input, parseAny(ctx, chunks, true))
	}

	return tags
}

func (d defaultDataParser) ParseStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) map[string]any {
	defer output.Close()

	if info == nil {
		return nil
	}

	tags := make(spanTags)

	switch info.Component {
	case components.ComponentOfChatModel:
		tags = d.ParseChatModelStreamOutput(ctx, output)

		tags.set(obtag.Stream, true)
		tags.set(obtag.ModelProvider, info.Type)

	default:
		chunks, recvErr := d.ParseDefaultStreamOutput(ctx, output)
		if recvErr != nil {
			return tags.setTags(getErrorTags(ctx, recvErr))
		}

		tags.set(obtag.Output, parseAny(ctx, chunks, true))
	}

	return tags
}

func (d defaultDataParser) ParseChatModelStreamOutput(ctx context.Context, output *schema.StreamReader[callbacks.CallbackOutput]) map[string]any {
	var (
		chunks  []*schema.Message
		onceSet bool
		tags    = make(spanTags)
		usage   *model.TokenUsage
	)

	for {
		item, recvErr := output.Recv()
		if recvErr != nil {
			if recvErr == io.EOF {
				break
			}

			return tags.setTags(getErrorTags(ctx, recvErr))
		}

		cbOutput := model.ConvCallbackOutput(item)
		if cbOutput == nil {
			continue
		}

		if cbOutput.Message != nil {
			chunks = append(chunks, cbOutput.Message)
		}

		if cbOutput.TokenUsage != nil {
			usage = &model.TokenUsage{
				PromptTokens:     cbOutput.TokenUsage.PromptTokens,
				CompletionTokens: cbOutput.TokenUsage.CompletionTokens,
				TotalTokens:      cbOutput.TokenUsage.TotalTokens,
			}
		}

		if cbOutput.Config != nil && !onceSet {
			onceSet = true

			if tv, ok := getTraceVariablesValue(ctx); ok {
				tags.set(obtag.LatencyFirstResp, time.Since(tv.startTime).Milliseconds())
			}
		}
	}

	if msg, concatErr := schema.ConcatMessages(chunks); concatErr != nil { // unexpected
		tags.set(obtag.Output, parseAny(ctx, chunks, true))
	} else {
		tags.set(obtag.Output, convertModelOutput(&model.CallbackOutput{Message: msg}))
	}

	if usage != nil {
		tags.set(obtag.Tokens, usage.TotalTokens).
			set(obtag.InputTokens, usage.PromptTokens).
			set(obtag.OutputTokens, usage.CompletionTokens)
	}

	return tags
}

func (d defaultDataParser) ParseDefaultStreamInput(ctx context.Context, input *schema.StreamReader[callbacks.CallbackInput]) (chunks []callbacks.CallbackInput, err error) {
	for {
		item, recvErr := input.Recv()
		if recvErr != nil {
			if recvErr == io.EOF {
				break
			}

			return chunks, recvErr
		}

		chunks = append(chunks, item)
	}

	return chunks, nil
}

func (d defaultDataParser) ParseDefaultStreamOutput(ctx context.Context, output *schema.StreamReader[callbacks.CallbackOutput]) (chunks []callbacks.CallbackOutput, err error) {
	for {
		item, recvErr := output.Recv()
		if recvErr != nil {
			if recvErr == io.EOF {
				break
			}

			return chunks, recvErr
		}

		chunks = append(chunks, item)
	}

	return chunks, nil
}

func parseAny(ctx context.Context, v any, bStream bool) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case []*schema.Message:
		return toJson(t, bStream)

	case *schema.Message:
		return toJson(t, bStream)
	case string:
		if bStream {
			return toJson(t, bStream)
		}
		return t
	case interface{ String() string }:
		if bStream {
			return toJson(t.String(), bStream)
		}
		return t.String()

	case map[string]any:
		return toJson(t, bStream)

	case []callbacks.CallbackInput:
		return parseAny(ctx, toAnySlice(t), bStream)

	case []callbacks.CallbackOutput:
		return parseAny(ctx, toAnySlice(t), bStream)

	case []any:
		if len(t) > 0 {
			if _, ok := t[0].(*schema.Message); ok {
				msgs := make([]*schema.Message, 0, len(t))
				for i := range t {
					msg, ok := t[i].(*schema.Message)
					if ok {
						msgs = append(msgs, msg)
					}
				}

				return parseAny(ctx, msgs, bStream)
			}
		}

		return toJson(t, bStream)

	default:
		return toJson(v, bStream)
	}
}

func toAnySlice[T any](src []T) []any {
	resp := make([]any, len(src))
	for i := range src {
		resp[i] = src[i]
	}

	return resp
}

// parseSpanTypeFromComponent 转换 component 到 fornax 可以识别的 span_type
// span_type 会影响到 fornax 界面的展示
// TODO:
//   - 当前框架相比于之前缺失的后续需要补齐, 当前按照`原来的字符串`处理
//   - compose 相关概念的 component 概念(Chain/Graph/...), 当前也先按照`原来的字符串`处理
func parseSpanTypeFromComponent(c components.Component) string {
	switch c {
	case components.ComponentOfPrompt:
		return "prompt"

	case components.ComponentOfChatModel:
		return "model"

	case components.ComponentOfEmbedding:
		return "embedding"

	case components.ComponentOfIndexer:
		return "store"

	case components.ComponentOfRetriever:
		return "retriever"

	case components.ComponentOfLoaderSplitter:
		return "loader"

	case components.ComponentOfTool:
		return "function"

	default:
		return string(c)
	}
}
