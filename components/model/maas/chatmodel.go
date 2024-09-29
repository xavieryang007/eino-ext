// Package maas implements chat model for ark runtime.
package maas

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/debug"
	"time"

	"github.com/volcengine/volcengine-go-sdk/service/arkruntime"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
	autils "github.com/volcengine/volcengine-go-sdk/service/arkruntime/utils"

	"code.byted.org/flow/eino/callbacks"
	fmodel "code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	sutils "code.byted.org/flow/eino/schema/utils"
	"code.byted.org/flow/eino/utils/safe"
	"code.byted.org/lang/gg/gptr"
)

var (
	defaultBaseURL = "https://ark.cn-beijing.volces.com/api/v3"
	defaultRegion  = "cn-beijing"
)

var (
	ErrEmptyResponse = errors.New("empty response from model")
)

type ChatModelConfig struct {
	// default: "https://ark.cn-beijing.volces.com/api/v3"
	BaseURL string `json:"base_url"`
	// default: "cn-beijing"
	Region string `json:"region"`

	// one of APIKey or AccessKey/SecretKey is required.
	APIKey    string `json:"api_key"`
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`

	// endpoint_id on ark platform.
	Model string `json:"model"`

	/* -- Parameters in request -- */
	MaxTokens         *int                  `json:"max_tokens,omitempty"`
	Temperature       *float32              `json:"temperature,omitempty"`
	TopP              *float32              `json:"top_p,omitempty"`
	Stream            *bool                 `json:"stream,omitempty"`
	Stop              []string              `json:"stop,omitempty"`
	FrequencyPenalty  *float32              `json:"frequency_penalty,omitempty"`
	LogitBias         map[string]int        `json:"logit_bias,omitempty"`
	LogProbs          *bool                 `json:"log_probs,omitempty"`
	TopLogProbs       *int                  `json:"top_log_probs,omitempty"`
	User              *string               `json:"user,omitempty"`
	PresencePenalty   *float32              `json:"presence_penalty,omitempty"`
	RepetitionPenalty *float32              `json:"repetition_penalty,omitempty"`
	N                 *int                  `json:"n,omitempty"`
	ResponseFormat    *model.ResponseFormat `json:"response_format,omitempty"`
	Timeout           time.Duration         `json:"timeout"`
}

func buildClient(config *ChatModelConfig) *arkruntime.Client {
	if len(config.BaseURL) == 0 {
		config.BaseURL = defaultBaseURL
	}
	if len(config.Region) == 0 {
		config.Region = defaultRegion
	}

	if len(config.APIKey) > 0 {
		return arkruntime.NewClientWithApiKey(config.APIKey,
			arkruntime.WithBaseUrl(config.BaseURL),
			arkruntime.WithRegion(config.Region),
			arkruntime.WithTimeout(config.Timeout))
	}

	return arkruntime.NewClientWithAkSk(config.AccessKey, config.SecretKey,
		arkruntime.WithBaseUrl(config.BaseURL),
		arkruntime.WithRegion(config.Region),
		arkruntime.WithTimeout(config.Timeout))
}

func NewChatModel(_ context.Context, config *ChatModelConfig) (*ChatModel, error) {
	if config == nil {
		config = &ChatModelConfig{}
	}
	client := buildClient(config)
	return &ChatModel{
		config: config,
		client: client,
	}, nil
}

type ChatModel struct {
	config *ChatModelConfig
	client *arkruntime.Client
	tools  []tool
}

func (cm *ChatModel) Generate(ctx context.Context, in []*schema.Message, opts ...fmodel.Option) (
	outMsg *schema.Message, err error) {

	var (
		cbm, cbmOK = callbacks.ManagerFromCtx(ctx)
	)

	defer func() {
		if err != nil && cbmOK {
			_ = cbm.OnError(ctx, err)
		}
	}()

	req, err := cm.genRequest(in, opts...)
	if err != nil {
		return nil, err
	}

	reqConf := &fmodel.Config{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &fmodel.CallbackInput{
			Messages: in,
			Config:   reqConf,
		})
	}

	resp, err := cm.client.CreateChatCompletion(ctx, *req)
	if err != nil {
		return nil, fmt.Errorf("[MaasV3] CreateChatCompletion error, %v", err)
	}

	outMsg, usage, err := cm.resolveChatResponse(resp)
	if err != nil {
		return nil, err
	}

	if cbmOK {
		_ = cbm.OnEnd(ctx, &fmodel.CallbackOutput{
			Message:    outMsg,
			Config:     reqConf,
			TokenUsage: usage,
		})
	}

	return outMsg, nil
}

func (cm *ChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...fmodel.Option) ( // byted_s_too_many_lines_in_func
	outStream *schema.StreamReader[*schema.Message], err error) {

	var (
		cbm, cbmOK = callbacks.ManagerFromCtx(ctx)
	)

	defer func() {
		if err != nil && cbmOK {
			_ = cbm.OnError(ctx, err)
		}
	}()

	req, err := cm.genRequest(in, opts...)
	if err != nil {
		return nil, err
	}
	req.Stream = true
	reqConf := &fmodel.Config{
		Model:       req.Model,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		TopP:        req.TopP,
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &fmodel.CallbackInput{
			Messages: in,
			Config:   reqConf,
		})
	}

	stream, err := cm.client.CreateChatCompletionStream(ctx, *req)
	if err != nil {
		return nil, err
	}

	rawStream := schema.NewStream[*fmodel.CallbackOutput](1)
	go func() {
		defer func() {
			panicErr := recover()

			if panicErr != nil {
				_ = rawStream.Send(nil, safe.NewPanicErr(panicErr, debug.Stack()))
			}

			rawStream.Finish()
			closeMaaSStreamReader(stream)

		}()

		for {
			resp, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				return
			}

			if err != nil {
				_ = rawStream.Send(nil, err)
				return
			}

			msg, usage, e := cm.resolveStreamResponse(resp)
			if e != nil {
				_ = rawStream.Send(nil, e)
				return
			}

			closed := rawStream.Send(&fmodel.CallbackOutput{
				Message:    msg,
				Config:     reqConf,
				TokenUsage: usage,
			}, nil)
			if closed {
				return
			}
		}
	}()

	rawStreamArr := make([]*schema.StreamReader[*fmodel.CallbackOutput], 2)
	if cbmOK {
		rawStreamArr = rawStream.AsReader().Copy(2)
	} else {
		rawStreamArr[0] = rawStream.AsReader()
	}

	outStream = schema.StreamReaderWithConvert(rawStreamArr[0],
		func(src *fmodel.CallbackOutput) (*schema.Message, error) {
			return src.Message, nil
		},
	)

	if cbmOK {
		cbStream := schema.StreamReaderWithConvert(rawStreamArr[1],
			func(src *fmodel.CallbackOutput) (callbacks.CallbackOutput, error) {
				return src, nil
			},
		)
		_ = cbm.OnEndWithStreamOutput(ctx, cbStream)
	}

	return outStream, nil
}

func (cm *ChatModel) genRequest(in []*schema.Message, opts ...fmodel.Option) (req *model.ChatCompletionRequest, err error) {
	options := fmodel.GetCommonOptions(&fmodel.Options{
		Temperature: cm.config.Temperature,
		MaxTokens:   cm.config.MaxTokens,
		Model:       &cm.config.Model,
		TopP:        cm.config.TopP,
	}, opts...)

	if options.Model == nil || len(*options.Model) == 0 {
		return nil, fmt.Errorf("maas chat model gen request with empty model")
	}

	req = &model.ChatCompletionRequest{
		MaxTokens:         dereferenceOrZero(options.MaxTokens),
		Temperature:       dereferenceOrZero(options.Temperature),
		TopP:              dereferenceOrZero(options.TopP),
		Model:             dereferenceOrZero(options.Model),
		Stream:            dereferenceOrZero(cm.config.Stream),
		Stop:              cm.config.Stop,
		FrequencyPenalty:  dereferenceOrZero(cm.config.FrequencyPenalty),
		LogitBias:         cm.config.LogitBias,
		LogProbs:          dereferenceOrZero(cm.config.LogProbs),
		TopLogProbs:       dereferenceOrZero(cm.config.TopLogProbs),
		User:              dereferenceOrZero(cm.config.User),
		PresencePenalty:   dereferenceOrZero(cm.config.PresencePenalty),
		RepetitionPenalty: dereferenceOrZero(cm.config.RepetitionPenalty),
		N:                 dereferenceOrZero(cm.config.N),
		ResponseFormat:    cm.config.ResponseFormat,
	}
	for _, msg := range in {
		content, e := toMaasContent(msg.Content, msg.MultiContent)
		if e != nil {
			return req, e
		}

		req.Messages = append(req.Messages, &model.ChatCompletionMessage{
			Content:    content,
			Role:       string(msg.Role),
			ToolCallID: msg.ToolCallID,
			ToolCalls:  toMaasToolCalls(msg.ToolCalls),
		})
	}

	req.Tools = make([]*model.Tool, 0, len(cm.tools))

	for _, tool := range cm.tools {
		maasTool := &model.Tool{
			Type: model.ToolTypeFunction,
			Function: &model.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}

		req.Tools = append(req.Tools, maasTool)
	}

	return req, nil
}

func (cm *ChatModel) resolveChatResponse(resp model.ChatCompletionResponse) (msg *schema.Message, usage *fmodel.TokenUsage, err error) {
	if len(resp.Choices) == 0 {
		return nil, nil, ErrEmptyResponse
	}

	choice := resp.Choices[0]

	content := choice.Message.Content
	if content == nil && len(choice.Message.ToolCalls) == 0 {
		return nil, nil, fmt.Errorf("unexpected message, nil content and no tool calls")
	}

	msg = &schema.Message{
		Role:       schema.RoleType(choice.Message.Role),
		ToolCallID: choice.Message.ToolCallID,
		ToolCalls:  toMessageToolCalls(choice.Message.ToolCalls),
	}

	if content.StringValue != nil {
		msg.Content = *content.StringValue
	}

	usage = &fmodel.TokenUsage{
		PromptTokens:     resp.Usage.PromptTokens,
		CompletionTokens: resp.Usage.CompletionTokens,
		TotalTokens:      resp.Usage.TotalTokens,
	}

	return msg, usage, nil
}

func (cm *ChatModel) resolveStreamResponse(resp model.ChatCompletionStreamResponse) (msg *schema.Message, usage *fmodel.TokenUsage, err error) {
	if len(resp.Choices) == 0 {
		return nil, nil, ErrEmptyResponse
	}

	choice := resp.Choices[0]

	err = getUnexpectedFinishReason(choice.FinishReason)
	if err != nil {
		return nil, nil, err
	}

	content := choice.Delta.Content

	msg = &schema.Message{
		Role:      schema.RoleType(choice.Delta.Role),
		ToolCalls: toMessageToolCalls(choice.Delta.ToolCalls),
		Content:   content,
	}

	if resp.Usage != nil {
		usage = &fmodel.TokenUsage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		}
	}

	return msg, usage, nil
}

func (cm *ChatModel) GetType() string {
	return getType()
}

func (cm *ChatModel) IsCallbacksEnabled() bool {
	return true
}

func (cm *ChatModel) BindTools(tools []*schema.ToolInfo) error {
	var err error
	cm.tools, err = toTools(tools)
	if err != nil {
		return err
	}

	return nil
}

func toMessageToolCalls(toolCalls []*model.ToolCall) []schema.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	ret := make([]schema.ToolCall, len(toolCalls))
	for i := range toolCalls {
		idx := i
		toolCall := toolCalls[i]
		ret[i] = schema.ToolCall{
			Index: &idx,
			ID:    toolCall.ID,
			Function: schema.FunctionCall{
				Name:      toolCall.Function.Name,
				Arguments: toolCall.Function.Arguments,
			},
		}
	}

	return ret
}

func toMaasContent(content string, multiContent []schema.ChatMessagePart) (*model.ChatCompletionMessageContent, error) {
	if len(multiContent) == 0 {
		return &model.ChatCompletionMessageContent{StringValue: gptr.Of(content)}, nil
	}

	parts := make([]*model.ChatCompletionMessageContentPart, 0, len(multiContent))

	for _, part := range multiContent {
		switch part.Type {
		case schema.ChatMessagePartTypeText:
			parts = append(parts, &model.ChatCompletionMessageContentPart{
				Type: model.ChatCompletionMessageContentPartTypeText,
				Text: part.Text,
			})
		case schema.ChatMessagePartTypeImageURL:
			parts = append(parts, &model.ChatCompletionMessageContentPart{
				Type: model.ChatCompletionMessageContentPartTypeImageURL,
				ImageURL: &model.ChatMessageImageURL{
					URL:    part.ImageURL.URL,
					Detail: model.ImageURLDetail(part.ImageURL.Detail),
				},
			})
		default:
			return nil, fmt.Errorf("unsupported chat message part type: %s", part.Type)
		}
	}

	return &model.ChatCompletionMessageContent{
		ListValue: parts,
	}, nil
}

func toMaasToolCalls(toolCalls []schema.ToolCall) []*model.ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	ret := make([]*model.ToolCall, len(toolCalls))
	for i := range toolCalls {
		toolCall := toolCalls[i]
		ret[i] = &model.ToolCall{
			ID:   toolCall.ID,
			Type: model.ToolTypeFunction,
			Function: model.FunctionCall{
				Arguments: toolCall.Function.Arguments,
				Name:      toolCall.Function.Name,
			},
		}
	}

	return ret
}

func toTools(tls []*schema.ToolInfo) ([]tool, error) {
	tools := make([]tool, len(tls))
	for i := range tls {
		ti := tls[i]
		if ti == nil {
			return nil, errors.New("unexpected nil tool")
		}

		paramsJSONSchema := sutils.ParamInfosToJSONSchema(ti.Params)

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

func closeMaaSStreamReader(r *autils.ChatCompletionStreamReader) error {
	if r == nil || r.Response == nil || r.Response.Body == nil {
		return nil
	}

	return r.Close()
}
