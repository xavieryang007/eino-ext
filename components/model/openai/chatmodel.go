// openai implement openai model as a chat model.
package openai

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/sashabaranov/go-openai"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/eino/utils/safe"

	"code.byted.org/flow/eino-ext/components/model/openai/internal/transport"
)

type ChatCompletionResponseFormatType string

const (
	ChatCompletionResponseFormatTypeJSONObject ChatCompletionResponseFormatType = "json_object"
	ChatCompletionResponseFormatTypeText       ChatCompletionResponseFormatType = "text"
)

const (
	toolChoiceRequired = "required"
)

type ChatCompletionResponseFormat struct {
	Type ChatCompletionResponseFormatType `json:"type,omitempty"`
}

type ChatModelConfig struct {
	BaseURL string `json:"base_url"`

	APIKey     string `json:"api_key"`
	ByAzure    bool   `json:"by_azure"`
	APIVersion string `json:"api_version"`

	Model string `json:"model"`

	MaxTokens       *int     `json:"max_tokens,omitempty"`
	Temperature     *float32 `json:"temperature,omitempty"`
	TopP            *float32 `json:"top_p,omitempty"`
	Stop            []string `json:"stop,omitempty"`
	PresencePenalty *float32 `json:"presence_penalty,omitempty"`
	// ResponseFormat is the format of the response.
	// can be "json_object" or "text".
	// tips: if you use "json_object", the prompt must contains a `JSON` string.
	// refs: https://platform.openai.com/docs/guides/structured-outputs/json-mode.
	ResponseFormat   *ChatCompletionResponseFormat `json:"response_format,omitempty"`
	Seed             *int                          `json:"seed,omitempty"`
	FrequencyPenalty *float32                      `json:"frequency_penalty,omitempty"`
	// LogitBias is must be a token id string (specified by their token ID in the tokenizer), not a word string.
	// incorrect: `"logit_bias":{"You": 6}`, correct: `"logit_bias":{"1639": 6}`
	// refs: https://platform.openai.com/docs/api-reference/chat/create#chat/create-logit_bias
	LogitBias map[string]int `json:"logit_bias,omitempty"`
	// LogProbs indicates whether to return log probabilities of the output tokens or not.
	// If true, returns the log probabilities of each output token returned in the content of message.
	// This option is currently not available on the gpt-4-vision-preview model.
	LogProbs *bool `json:"logprobs,omitempty"`
	// TopLogProbs is an integer between 0 and 5 specifying the number of most likely tokens to return at each
	// token position, each with an associated log probability.
	// logprobs must be set to true if this parameter is used.
	TopLogProbs *int          `json:"top_logprobs,omitempty"`
	User        *string       `json:"user,omitempty"`
	Timeout     time.Duration `json:"timeout"`
}

var _ model.ChatModel = (*ChatModel)(nil)

type ChatModel struct {
	cli    *openai.Client
	config *ChatModelConfig

	tools         []tool
	rawTools      []*schema.ToolInfo
	forceToolCall bool
}

func NewChatModel(ctx context.Context, config *ChatModelConfig) (*ChatModel, error) {
	if config == nil {
		config = &ChatModelConfig{Model: "gpt-3.5-turbo"}
	}

	var clientConf openai.ClientConfig

	if config.ByAzure {
		clientConf = openai.DefaultAzureConfig(config.APIKey, config.BaseURL)
		if config.APIVersion != "" {
			clientConf.APIVersion = config.APIVersion
		}
	} else {
		clientConf = openai.DefaultConfig(config.APIKey)
		if len(config.BaseURL) > 0 {
			clientConf.BaseURL = config.BaseURL
		}
	}

	clientConf.HTTPClient = &http.Client{
		Timeout:   config.Timeout,
		Transport: &transport.HeaderTransport{Origin: http.DefaultTransport},
	}

	return &ChatModel{
		cli:    openai.NewClientWithConfig(clientConf),
		config: config,
	}, nil
}

func toOpenAIRole(role schema.RoleType) string {
	switch role {
	case schema.User:
		return openai.ChatMessageRoleUser
	case schema.Assistant:
		return openai.ChatMessageRoleAssistant
	case schema.System:
		return openai.ChatMessageRoleSystem
	case schema.Tool:
		return openai.ChatMessageRoleTool
	default:
		return string(role)
	}
}

func toOpenAIMultiContent(mc []schema.ChatMessagePart) ([]openai.ChatMessagePart, error) {
	if len(mc) == 0 {
		return nil, nil
	}

	ret := make([]openai.ChatMessagePart, 0, len(mc))

	for _, part := range mc {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			ret = append(ret, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: part.Text,
			})
		case schema.ChatMessagePartTypeImageURL:
			if part.ImageURL == nil {
				return nil, fmt.Errorf("image_url should not be nil")
			}
			ret = append(ret, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				ImageURL: &openai.ChatMessageImageURL{
					URL:    part.ImageURL.URL,
					Detail: openai.ImageURLDetail(part.ImageURL.Detail),
				},
			})
		default:
			return nil, fmt.Errorf("unsupported chat message part type: %s", part.Type)
		}
	}

	return ret, nil
}

func toMessageRole(role string) schema.RoleType {
	switch role {
	case openai.ChatMessageRoleUser:
		return schema.User
	case openai.ChatMessageRoleAssistant:
		return schema.Assistant
	case openai.ChatMessageRoleSystem:
		return schema.System
	case openai.ChatMessageRoleTool:
		return schema.Tool
	default:
		return schema.RoleType(role)
	}
}

func toMessageToolCalls(toolCalls []openai.ToolCall) []schema.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	ret := make([]schema.ToolCall, len(toolCalls))
	for i := range toolCalls {
		toolCall := toolCalls[i]
		ret[i] = schema.ToolCall{
			Index: toolCall.Index,
			ID:    toolCall.ID,
			Function: schema.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}
	}

	return ret
}

func toOpenAIToolCalls(toolCalls []schema.ToolCall) []openai.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	ret := make([]openai.ToolCall, len(toolCalls))
	for i := range toolCalls {
		toolCall := toolCalls[i]
		ret[i] = openai.ToolCall{
			Index: toolCall.Index,
			ID:    toolCall.ID,
			Type:  openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}
	}

	return ret
}

func (cm *ChatModel) genRequest(in []*schema.Message, options *model.Options) (*openai.ChatCompletionRequest, error) {
	if options.Model == nil || len(*options.Model) == 0 {
		return nil, fmt.Errorf("open chat model gen request with empty model")
	}

	req := &openai.ChatCompletionRequest{
		Model:            *options.Model,
		MaxTokens:        dereferenceOrZero(options.MaxTokens),
		Temperature:      dereferenceOrZero(options.Temperature),
		TopP:             dereferenceOrZero(options.TopP),
		Stop:             cm.config.Stop,
		PresencePenalty:  dereferenceOrZero(cm.config.PresencePenalty),
		Seed:             cm.config.Seed,
		FrequencyPenalty: dereferenceOrZero(cm.config.FrequencyPenalty),
		LogitBias:        cm.config.LogitBias,
		LogProbs:         dereferenceOrZero(cm.config.LogProbs),
		TopLogProbs:      dereferenceOrZero(cm.config.TopLogProbs),
		User:             dereferenceOrZero(cm.config.User),
	}

	if len(cm.tools) > 0 {
		req.Tools = make([]openai.Tool, len(cm.tools))
		for i := range cm.tools {
			t := cm.tools[i]

			req.Tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        t.Function.Name,
					Description: t.Function.Description,
					Parameters:  t.Function.Parameters,
				},
			}
		}

		if cm.forceToolCall && len(cm.tools) > 0 {

			/* // nolint: byted_s_comment_space
			tool_choice is string or object
			Controls which (if any) tool is called by the model.
			"none" means the model will not call any tool and instead generates a message.
			"auto" means the model can pick between generating a message or calling one or more tools.
			"required" means the model must call one or more tools.

			Specifying a particular tool via {"type": "function", "function": {"name": "my_function"}} forces the model to call that tool.

			"none" is the default when no tools are present.
			"auto" is the default if tools are present.
			*/

			if len(req.Tools) > 1 {
				req.ToolChoice = toolChoiceRequired
			} else {
				req.ToolChoice = openai.ToolChoice{
					Type: req.Tools[0].Type,
					Function: openai.ToolFunction{
						Name: req.Tools[0].Function.Name,
					},
				}
			}
		}
	}

	msgs := make([]openai.ChatCompletionMessage, 0, len(in))
	for _, inMsg := range in {
		mc, e := toOpenAIMultiContent(inMsg.MultiContent)
		if e != nil {
			return nil, e
		}
		msg := openai.ChatCompletionMessage{
			Role:         toOpenAIRole(inMsg.Role),
			Content:      inMsg.Content,
			MultiContent: mc,
			Name:         inMsg.Name,
			ToolCalls:    toOpenAIToolCalls(inMsg.ToolCalls),
			ToolCallID:   inMsg.ToolCallID,
		}

		msgs = append(msgs, msg)
	}

	req.Messages = msgs

	if cm.config.ResponseFormat != nil {
		req.ResponseFormat = &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatType(cm.config.ResponseFormat.Type),
		}
	}

	return req, nil
}

func (cm *ChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...model.Option) (
	outMsg *schema.Message, err error) {

	var (
		cbm, cbmOK = callbacks.ManagerFromCtx(ctx)
	)

	defer func() {
		if err != nil && cbmOK {
			_ = cbm.OnError(ctx, err)
		}
	}()

	options := model.GetCommonOptions(&model.Options{
		Temperature: cm.config.Temperature,
		MaxTokens:   cm.config.MaxTokens,
		Model:       &cm.config.Model,
		TopP:        cm.config.TopP,
		Stop:        cm.config.Stop,
	}, opts...)

	req, err := cm.genRequest(in, options)
	if err != nil {
		return nil, err
	}

	reqConf := &model.Config{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &model.CallbackInput{
			Messages:   in,
			Tools:      append(cm.rawTools), // join tool info from call options
			ToolChoice: getToolChoice(req.ToolChoice),
			Config:     reqConf,
		})
	}

	resp, err := cm.cli.CreateChatCompletion(ctx, *req)
	if err != nil {
		return nil, err
	}

	msg := resp.Choices[0].Message

	outMsg = &schema.Message{
		Role:       toMessageRole(msg.Role),
		Content:    msg.Content,
		Name:       msg.Name,
		ToolCallID: msg.ToolCallID,
		ToolCalls:  toMessageToolCalls(msg.ToolCalls),
	}

	usage := &model.TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	if cbmOK {
		_ = cbm.OnEnd(ctx, &model.CallbackOutput{
			Message:    outMsg,
			Config:     reqConf,
			TokenUsage: usage,
		})
	}

	return outMsg, nil
}

func (cm *ChatModel) Stream(ctx context.Context, in []*schema.Message, // nolint: byted_s_too_many_lines_in_func
	opts ...model.Option) (outStream *schema.StreamReader[*schema.Message], err error) {

	var (
		cbm, cbmOK = callbacks.ManagerFromCtx(ctx)
	)

	defer func() {
		if err != nil && cbmOK {
			_ = cbm.OnError(ctx, err)
		}
	}()

	options := model.GetCommonOptions(&model.Options{
		Temperature: cm.config.Temperature,
		MaxTokens:   cm.config.MaxTokens,
		Model:       &cm.config.Model,
		TopP:        cm.config.TopP,
		Stop:        cm.config.Stop,
	}, opts...)

	req, err := cm.genRequest(in, options)
	if err != nil {
		return nil, err
	}

	req.Stream = true

	reqConf := &model.Config{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
		Stop:        req.Stop,
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &model.CallbackInput{
			Messages:   in,
			Tools:      append(cm.rawTools), // join tool info from call options
			ToolChoice: getToolChoice(req.ToolChoice),
			Config:     reqConf,
		})
	}

	stream, err := cm.cli.CreateChatCompletionStream(ctx, *req)
	if err != nil {
		return nil, err
	}

	rawStream := schema.NewStream[*model.CallbackOutput](1)
	go func() {
		defer func() {
			panicErr := recover()
			_ = stream.Close()

			if panicErr != nil {
				_ = rawStream.Send(nil, safe.NewPanicErr(panicErr, debug.Stack()))
			}

			rawStream.Finish()
		}()

		var lastEmptyMsg *schema.Message

		for {
			chunk, chunkErr := stream.Recv()
			if chunkErr == io.EOF {
				return
			}

			if chunkErr != nil {
				_ = rawStream.Send(nil, chunkErr)
				return
			}

			delta := chunk.Choices[0].Delta
			msg := &schema.Message{
				Role:      toMessageRole(delta.Role),
				Content:   delta.Content,
				ToolCalls: toMessageToolCalls(delta.ToolCalls),
			}

			// skip empty message
			// when openai return parallel tool calls, first frame can be empty
			// skip empty frame in stream, then stream first frame could know whether is tool call msg.
			if lastEmptyMsg != nil {
				cMsg, cErr := schema.ConcatMessages([]*schema.Message{lastEmptyMsg, msg})
				if cErr != nil { // nolint: byted_s_too_many_nests_in_func
					_ = rawStream.Send(nil, cErr)
					return
				}

				msg = cMsg
			}

			if msg.Content == "" && len(msg.ToolCalls) == 0 {
				lastEmptyMsg = msg
				continue
			}

			lastEmptyMsg = nil

			var tokenUsage *model.TokenUsage
			if chunk.Usage != nil {
				tokenUsage = &model.TokenUsage{
					PromptTokens:     chunk.Usage.PromptTokens,
					CompletionTokens: chunk.Usage.CompletionTokens,
					TotalTokens:      chunk.Usage.TotalTokens,
				}
			}

			closed := rawStream.Send(&model.CallbackOutput{
				Message:    msg,
				Config:     reqConf,
				TokenUsage: tokenUsage,
			}, nil)

			if closed {
				return
			}
		}
	}()

	rawStreamArr := make([]*schema.StreamReader[*model.CallbackOutput], 2)
	if cbmOK {
		rawStreamArr = rawStream.AsReader().Copy(2) // nolint: byted_s_magic_number
	} else {
		rawStreamArr[0] = rawStream.AsReader()
	}

	outStream = schema.StreamReaderWithConvert(rawStreamArr[0],
		func(src *model.CallbackOutput) (*schema.Message, error) {
			return src.Message, nil
		})

	if cbmOK {
		cbStream := schema.StreamReaderWithConvert(rawStreamArr[1],
			func(src *model.CallbackOutput) (callbacks.CallbackOutput, error) {
				return src, nil
			},
		)
		_ = cbm.OnEndWithStreamOutput(ctx, cbStream)
	}

	return outStream, nil
}

func toTools(tis []*schema.ToolInfo) ([]tool, error) {
	tools := make([]tool, len(tis))
	for i := range tis {
		ti := tis[i]
		if ti == nil {
			return nil, errors.New("unexpected nil tool")
		}

		paramsJSONSchema, err := ti.ParamsOneOf.ToOpenAPIV3()
		if err != nil {
			return nil, fmt.Errorf("convert toolInfo ParamsOneOf to JSONSchema failed: %w", err)
		}

		tools[i] = tool{
			Function: &functionDefinition{
				Name:        ti.Name,
				Description: ti.Desc,
				Parameters:  paramsJSONSchema,
			},
		}
	}

	return tools, nil
}

func (cm *ChatModel) BindTools(tools []*schema.ToolInfo) error {
	var err error
	cm.tools, err = toTools(tools)
	if err != nil {
		return err
	}

	cm.forceToolCall = false
	cm.rawTools = tools

	return nil
}

func (cm *ChatModel) BindForcedTools(tools []*schema.ToolInfo) error {
	var err error
	cm.tools, err = toTools(tools)
	if err != nil {
		return err
	}

	cm.forceToolCall = true
	cm.rawTools = tools

	return nil
}

func (cm *ChatModel) GetType() string {
	return typ
}

func (cm *ChatModel) IsCallbacksEnabled() bool {
	return true
}

func getToolChoice(choice any) any {
	switch t := choice.(type) {
	case string:
		return t
	case openai.ToolChoice:
		return &schema.ToolInfo{
			Name: t.Function.Name,
		}
	default:
		return nil
	}
}
