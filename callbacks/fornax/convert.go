package fornax

import (
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/components/prompt"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/flow-telemetry-common/go/obtype"
	"code.byted.org/gopkg/logs/v2"
)

const (
	toolTypeFunction = "function"
)

// ChatModel

func convertModelInput(input *model.CallbackInput) *obtype.ModelInput {
	return &obtype.ModelInput{
		Messages:        iter(input.Messages, convertModelMessage),
		Tools:           iter(input.Tools, convertTool),
		ModelToolChoice: convertToolChoice(input.ToolChoice),
	}
}

func convertModelOutput(output *model.CallbackOutput) *obtype.ModelOutput {
	return &obtype.ModelOutput{
		Choices: []*obtype.ModelChoice{
			{Index: 0, Message: convertModelMessage(output.Message)},
		},
	}
}

func convertModelMessage(message *schema.Message) *obtype.ModelMessage {
	if message == nil {
		return nil
	}

	msg := &obtype.ModelMessage{
		Role:       string(message.Role),
		Content:    message.Content,
		Parts:      make([]*obtype.ModelMessagePart, len(message.MultiContent)),
		Name:       message.Name,
		ToolCalls:  make([]*obtype.ModelToolCall, len(message.ToolCalls)),
		ToolCallID: message.ToolCallID,
	}

	for i := range message.MultiContent {
		part := message.MultiContent[i]

		msg.Parts[i] = &obtype.ModelMessagePart{
			Type: string(part.Type),
			Text: part.Text,
		}

		if part.ImageURL != nil {
			msg.Parts[i].ImageURL = &obtype.ModelImageURL{
				URL:    part.ImageURL.URL,
				Detail: string(part.ImageURL.Detail),
			}
		}
	}

	for i := range message.ToolCalls {
		tc := message.ToolCalls[i]

		msg.ToolCalls[i] = &obtype.ModelToolCall{
			ID:   tc.ID,
			Type: toolTypeFunction,
			Function: &obtype.ModelToolCallFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}

	return msg
}

func convertTool(tool *schema.ToolInfo) *obtype.ModelTool {
	if tool == nil {
		return nil
	}

	var params []byte
	if raw, err := tool.ParamsOneOf.ToOpenAPIV3(); err != nil {
		logs.Warnf("[convertTool] param ToOpenAPIV3 failed, err=%v", err)
	} else {
		params, err = raw.MarshalJSON()
		if err != nil {
			logs.Warnf("[convertTool] marshal openapi3.Schema failed, err=%v", err)
		}
	}

	t := &obtype.ModelTool{
		Type: toolTypeFunction,
		Function: &obtype.ModelToolFunction{
			Name:        tool.Name,
			Description: tool.Desc,
			Parameters:  params,
		},
	}

	return t
}

func convertToolChoice(choice any) *obtype.ModelToolChoice {
	if choice == nil {
		return nil
	}

	switch t := choice.(type) {
	case string:
		return &obtype.ModelToolChoice{
			Type: t,
		}

	case *schema.ToolInfo:
		return &obtype.ModelToolChoice{
			Function: &obtype.ModelToolCallFunction{
				Name: t.Name,
			},
		}
	default:
		return nil
	}
}

func convertModelCallOption(config *model.Config) *obtype.ModelCallOption {
	if config == nil {
		return nil
	}

	return &obtype.ModelCallOption{
		Temperature: config.Temperature,
		MaxTokens:   int64(config.MaxTokens),
		TopP:        config.TopP,
	}
}

// Prompt

func convertPromptInput(input *prompt.CallbackInput) *obtype.PromptInput {
	if input == nil {
		return nil
	}

	return &obtype.PromptInput{
		Templates: iter(input.Templates, convertTemplate),
		Arguments: convertPromptArguments(input.Variables),
	}
}

func convertPromptOutput(output *prompt.CallbackOutput) *obtype.PromptOutput {
	if output == nil {
		return nil
	}

	return &obtype.PromptOutput{
		Prompts: iter(output.Result, convertModelMessage),
	}
}

func convertTemplate(template schema.MessagesTemplate) *obtype.ModelMessage {
	if template == nil {
		return nil
	}

	switch t := template.(type) {
	case *schema.Message:
		return convertModelMessage(t)
	default: // messagePlaceholder etc.
		return nil
	}
}

func convertPromptArguments(variables map[string]any) []*obtype.PromptArgument {
	if variables == nil {
		return nil
	}

	resp := make([]*obtype.PromptArgument, 0, len(variables))

	for k := range variables {
		resp = append(resp, &obtype.PromptArgument{
			Key:   k,
			Value: variables[k],
			// Source: "",
		})
	}

	return resp
}

// Retriever

func convertRetrieverOutput(output *retriever.CallbackOutput) *obtype.RetrieverOutput {
	if output == nil {
		return nil
	}

	return &obtype.RetrieverOutput{
		Documents: iter(output.Docs, convertDocument),
	}
}

func convertRetrieverCallOption(input *retriever.CallbackInput) *obtype.RetrieverCallOption {
	if input == nil {
		return nil
	}

	opt := &obtype.RetrieverCallOption{
		TopK:   int64(input.TopK),
		Filter: input.Filter,
	}

	if input.ScoreThreshold != nil {
		opt.MinScore = input.ScoreThreshold
	}

	return opt
}

func convertDocument(doc *schema.Document) *obtype.RetrieverDocument {
	if doc == nil {
		return nil
	}

	return &obtype.RetrieverDocument{
		ID:      doc.ID,
		Content: doc.Content,
		Score:   doc.Score(),
		// Index:   "",
		// Vector:  nil,
	}
}

func iter[A, B any](sa []A, fb func(a A) B) []B {
	r := make([]B, len(sa))
	for i := range sa {
		r[i] = fb(sa[i])
	}

	return r
}
