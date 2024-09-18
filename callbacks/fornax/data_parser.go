package fornax

import (
	"context"
	"io"

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
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/lang/gg/gslice"
)

// CallbackDataParser tag parser for trace
// Implement CallbackDataParser and replace defaultDataParser by WithCallbackDataParser if needed
type CallbackDataParser interface {
	ParseInput(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) map[string]any
	ParseOutput(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) map[string]any
	ParseStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) map[string]any
	ParseStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) map[string]any
}

type defaultDataParser struct{}

func (d defaultDataParser) ParseInput(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) map[string]any {
	tags := make(spanTags)

	switch info.Component {
	case components.ComponentOfChatModel:
		cbInput := model.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, parseAny(ctx, cbInput.Messages))
		}

		if cbInput.Config != nil {
			tags.set(obtag.ModelName, cbInput.Config.Model)
		}

	case components.ComponentOfPrompt:
		cbInput := prompt.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, parseAny(ctx, cbInput.Variables))
		}

	case components.ComponentOfEmbedding:
		cbInput := embedding.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, cbInput.Texts)
		}

		if cbInput.Config != nil {
			tags.set(obtag.ModelName, cbInput.Config.Model)
		}

	case components.ComponentOfRetriever:
		cbInput := retriever.ConvCallbackInput(input)
		if cbInput != nil {
			tags.set(obtag.Input, parseAny(ctx, cbInput.Query))
		}

	case components.ComponentOfIndexer:
		cbInput := indexer.ConvCallbackInput(input)
		if cbInput != nil {
			// rewrite if not suitable here
			tags.set(obtag.Input, parseAny(ctx, cbInput.Docs))
		}

	case compose.ComponentOfLambda:
		tags.set(obtag.Input, parseAny(ctx, input))

	default:
		tags.set(obtag.Input, parseAny(ctx, input))
	}

	return tags
}

func (d defaultDataParser) ParseOutput(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) map[string]any {
	tags := make(spanTags)

	switch info.Component {
	case components.ComponentOfChatModel:
		cbOutput := model.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, parseAny(ctx, cbOutput.Message))
		}

		if cbOutput.TokenUsage != nil {
			tags.set(obtag.Tokens, cbOutput.TokenUsage.TotalTokens).
				set(obtag.InputTokens, cbOutput.TokenUsage.PromptTokens).
				set(obtag.OutputTokens, cbOutput.TokenUsage.CompletionTokens)
		}

		if cbOutput.Config != nil {
			tags.set(obtag.ModelName, cbOutput.Config.Model)
		}
	case components.ComponentOfPrompt:
		cbOutput := prompt.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, parseAny(ctx, cbOutput.Result))
		}

	case components.ComponentOfEmbedding:
		cbOutput := embedding.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, parseAny(ctx, cbOutput.Embeddings))
		}

		if cbOutput.TokenUsage != nil {
			tags.set(obtag.Tokens, cbOutput.TokenUsage.TotalTokens).
				set(obtag.InputTokens, cbOutput.TokenUsage.PromptTokens).
				set(obtag.OutputTokens, cbOutput.TokenUsage.CompletionTokens)
		}

		if cbOutput.Config != nil {
			tags.set(obtag.ModelName, cbOutput.Config.Model)
		}
	case components.ComponentOfIndexer:
		cbOutput := indexer.ConvCallbackOutput(output)
		if cbOutput != nil {
			tags.set(obtag.Output, parseAny(ctx, cbOutput.IDs))
		}

	case components.ComponentOfRetriever:
		cbOutput := retriever.ConvCallbackOutput(output)
		if cbOutput != nil {
			// rewrite if not suitable here
			tags.set(obtag.Output, parseAny(ctx, cbOutput.Docs))
		}

	case compose.ComponentOfLambda:
		tags.set(obtag.Output, parseAny(ctx, output))

	default:
		tags.set(obtag.Output, parseAny(ctx, output))

	}

	return tags
}

func (d defaultDataParser) ParseStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) map[string]any {
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

		tags.set(obtag.Input, parseAny(ctx, chunks))
	}

	return tags
}

func (d defaultDataParser) ParseStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) map[string]any {
	if info == nil {
		return nil
	}

	tags := make(spanTags)

	switch info.Component {
	case components.ComponentOfChatModel:
		tags = d.ParseChatModelStreamOutput(ctx, output)
	default:
		chunks, recvErr := d.ParseDefaultStreamOutput(ctx, output)
		if recvErr != nil {
			return tags.setTags(getErrorTags(ctx, recvErr))
		}

		tags.set(obtag.Output, parseAny(ctx, chunks))
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

		chunks = append(chunks, cbOutput.Message)

		if cbOutput.TokenUsage != nil {
			if usage == nil {
				usage = &model.TokenUsage{}
			}

			usage.PromptTokens += cbOutput.TokenUsage.PromptTokens
			usage.TotalTokens += cbOutput.TokenUsage.TotalTokens
			usage.CompletionTokens += cbOutput.TokenUsage.CompletionTokens
		}

		if cbOutput.Config != nil && !onceSet {
			onceSet = true
			tags.set(obtag.ModelName, cbOutput.Config.Model)
		}
	}

	tags.set(obtag.Output, parseAny(ctx, chunks))

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

func parseAny(ctx context.Context, v any) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case []*schema.Message:
		var mgr []*schema.Message

		r := len(t) - 1
		for l := len(t) - 1; l >= 0; l-- {
			if t[l].Role != t[r].Role {
				var (
					msg *schema.Message
					err error
				)

				if t[r].Role == "" {
					msg, err = schema.ConcatMessages(t[l : r+1])
					r = l - 1
				} else if r-l == 1 {
					msg = t[r]
					r = l
				} else {
					msg, err = schema.ConcatMessages(t[l+1 : r+1])
					r = l
				}

				if err != nil {
					logs.CtxError(ctx, "[parseAny] parse []*schema.Message failed, err=%v", err)
					return ""
				}

				mgr = append(mgr, msg)
			}
		}

		if r >= 0 {
			msg, err := schema.ConcatMessages(t[:r+1])
			if err != nil {
				logs.CtxError(ctx, "[parseAny] parse []*schema.Message failed, err=%v", err)
				return ""
			}

			mgr = append(mgr, msg)
		}

		return toJson(gslice.ReverseClone(mgr))
	case *schema.Message:
		return toJson(t)
	case interface{ String() string }:
		return t.String()
	case map[string]any:
		return toJson(t)
	default:
		return toJson(v)
	}
}
