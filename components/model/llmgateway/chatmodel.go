package llmgateway

import (
	"context"
	"errors"
	"fmt"
	"io"
	"runtime/debug"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/model"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flow/eino/utils/safe"
	"code.byted.org/lang/gg/gptr"
	"code.byted.org/lang/gg/optional"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
	llm_gateway "code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway/llmgatewayservice"

	"code.byted.org/flow/eino-ext/components/model/llmgateway/internal/utils"
)

const (
	PSM = "stone.llm.gateway"
	TYP = "LLMGateway"
)

var (
	errEmptyResp = errors.New("empty response from model")
)

// ChatModelConfig Instance configs.
type ChatModelConfig struct {
	Model       string   `json:"model,omitempty"` // model_id
	MaxTokens   *int     `json:"max_tokens,omitempty"`
	Temperature *float32 `json:"temperature,omitempty"`
	TopP        *float32 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`

	PresencePenalty  *float32  `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float32  `json:"frequency_penalty,omitempty"`
	ResponseFormat   *string   `json:"response_format,omitempty"`
	Stop             *[]string `json:"stop,omitempty"`
	AutoFix          *bool     `json:"auto_fix,omitempty"`
	AK               *string   `json:"ak,omitempty"`
	SK               *string   `json:"sk,omitempty"`

	ToolChoice *gateway.ToolChoiceConfig `json:"tool_choice,omitempty"`
}

type ChatModel struct {
	config *ChatModelConfig
	client llm_gateway.StreamClient
	tools  []*gateway.Tool
}

func NewChatModel(_ context.Context, config *ChatModelConfig) (*ChatModel, error) {
	cli, err := llm_gateway.NewStreamClient(PSM)
	return &ChatModel{config: config, client: cli}, err
}

func (cm *ChatModel) BindTools(tools []*schema.ToolInfo) error {
	cm.tools = utils.ToGWTools(tools)
	return nil
}

func (cm *ChatModel) GetType() string {
	return TYP
}

func (cm *ChatModel) IsCallbacksEnabled() bool {
	return true
}

// Generate One shot generation.
func (cm *ChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	stm, err := cm.stream(ctx, false, input, opts...)
	if err != nil {
		return nil, err
	}

	defer stm.Close()
	msgs := make([]*schema.Message, 0)

	for {
		msg, err := stm.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	if len(msgs) == 0 {
		return nil, errEmptyResp
	}
	if len(msgs) == 1 {
		return msgs[0], nil
	}
	return schema.ConcatMessages(msgs)
}

// Stream Streaming interface.
func (cm *ChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (
	*schema.StreamReader[*schema.Message], error) {
	return cm.stream(ctx, true, input, opts...)
}

func (cm *ChatModel) stream(ctx context.Context, streamMode bool, input []*schema.Message, // nolint: byted_s_too_many_lines_in_func
	opts ...model.Option) (outStream *schema.StreamReader[*schema.Message], err error) {
	cbm, cbmOK := callbacks.ManagerFromCtx(ctx)
	defer func() {
		if err != nil && cbmOK {
			_ = cbm.OnError(ctx, err)
		}
	}()

	options, config, err := cm.genOptions(opts...)
	if err != nil {
		return nil, err
	}

	if cbmOK {
		ctx = cbm.OnStart(ctx, &model.CallbackInput{
			Messages: input,
			Config:   config,
		})
	}

	req, err := cm.genRequest(streamMode, options, input, opts...)
	if err != nil {
		return nil, err
	}

	stream, err := cm.client.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	sr, sw := schema.Pipe[*model.CallbackOutput](1)
	go func() {
		defer func() {
			panicErr := recover()
			_ = stream.Close()

			if panicErr != nil {
				_ = sw.Send(nil, safe.NewPanicErr(panicErr, debug.Stack()))
			}
			sw.Close()
		}()

		var lastEmptyMsg *schema.Message

		for {
			chunk, err := stream.Recv()
			if err != nil {
				if !errors.Is(err, io.EOF) {
					_ = sw.Send(nil, err)
				}
				return
			}

			msg, usage, rErr := cm.genResponse(chunk)
			if rErr != nil {
				_ = sw.Send(nil, rErr)
				return
			}

			if usage != nil {
				// stream usage return in last chunk without message content, then
				// last message received from callback output stream: Message == nil and TokenUsage != nil
				// last message received from outStream: Message != nil
				if closed := sw.Send(&model.CallbackOutput{
					Message:    msg,
					Config:     config,
					TokenUsage: usage,
				}, nil); closed {
					return
				}

				continue
			}

			if msg == nil && usage == nil {
				continue
			}

			// skip empty message
			// when openai return parallel tool calls, first frame can be empty
			// skip empty frame in stream, then stream first frame could know whether is tool call msg.
			if lastEmptyMsg != nil {
				cMsg, cErr := schema.ConcatMessages([]*schema.Message{lastEmptyMsg, msg})
				if cErr != nil { // nolint: byted_s_too_many_nests_in_func
					_ = sw.Send(nil, cErr)
					return
				}

				msg = cMsg
			}

			if msg.Content == "" && len(msg.ToolCalls) == 0 && len(msg.Extra) == 0 {
				lastEmptyMsg = msg
				continue
			}

			lastEmptyMsg = nil

			closed := sw.Send(&model.CallbackOutput{
				Message: msg,
				Config:  config,
			}, nil)

			if closed {
				return
			}
		}
	}()

	rawStreamArr := make([]*schema.StreamReader[*model.CallbackOutput], 2)
	if cbmOK {
		rawStreamArr = sr.Copy(2) // nolint: byted_s_magic_number
	} else {
		rawStreamArr[0] = sr
	}

	outStream = schema.StreamReaderWithConvert(rawStreamArr[0],
		func(src *model.CallbackOutput) (*schema.Message, error) {
			if src.Message == nil {
				return nil, schema.ErrNoValue
			}
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

func (cm *ChatModel) genResponse(resp *gateway.ChatCompletion) (msg *schema.Message, usg *model.TokenUsage, err error) {
	if resp == nil {
		return nil, nil, nil
	}

	if resp.Error != nil {
		return nil, nil, &Error{
			Message: resp.Error.Message,
			Code:    resp.Error.Code,
		}
	}
	msg = utils.ToEinoMessage(resp)
	usg = utils.ToEinoUsage(resp.Usage)
	return
}

func (cm *ChatModel) genOptions(opts ...model.Option) (*model.Options, *model.Config, error) {
	options := model.GetCommonOptions(&model.Options{
		Temperature: cm.config.Temperature,
		MaxTokens:   cm.config.MaxTokens,
		TopP:        cm.config.TopP,
		Model:       gptr.Of(cm.config.Model),
	}, opts...)

	if options.Model == nil || len(*options.Model) == 0 {
		return nil, nil, fmt.Errorf("request without specified model")
	}

	return options, &model.Config{
		Model:       *options.Model,
		MaxTokens:   optional.OfPtr(options.MaxTokens).ValueOr(0),
		Temperature: optional.OfPtr(options.Temperature).ValueOr(0),
		TopP:        optional.OfPtr(options.TopP).ValueOr(0),
	}, nil
}

func (cm *ChatModel) genRequest(stream bool, options *model.Options, messages []*schema.Message,
	opts ...model.Option) (*gateway.ChatRequest, error) {
	// Resolve customized llm_gateway request options.
	gwOpts := model.GetImplSpecificOptions[gatewayOptions](&gatewayOptions{}, opts...)
	if gwOpts.preProcessor != nil {
		if msgs, err := gwOpts.preProcessor(messages); err != nil {
			return nil, err
		} else {
			messages = msgs
		}
	}

	gwMsgs, err := utils.ToGWMessages(messages)
	if err != nil {
		return nil, err
	}

	// Init gateway chat request.
	req := &gateway.ChatRequest{
		ModelId:     *options.Model,
		Messages:    gwMsgs,
		UserInfo:    gwOpts.userInfo,
		Extra:       gwOpts.extra,
		ChatOptions: gwOpts.chatOptions,
		Traffic:     gwOpts.traffic,
		Tools:       cm.tools,
		Arguments: &gateway.Arguments{
			AutoFix: cm.config.AutoFix,
		},
	}

	req.ModelConfig = &gateway.ModelConfig{
		Stream: gptr.Of(stream),
		// Init model config by resolved options
		MaxTokens:   utils.I64Ptr(options.MaxTokens),
		Temperature: utils.F64Ptr(options.Temperature),
		TopP:        utils.F64Ptr(options.TopP),

		// Init model config by original chat model params
		TopK:             utils.I64Ptr(cm.config.TopK),
		FrequencyPenalty: utils.F64Ptr(cm.config.FrequencyPenalty),
		PresencePenalty:  utils.F64Ptr(cm.config.PresencePenalty),

		ToolChoice: cm.config.ToolChoice,
		Ak:         cm.config.AK,
		Sk:         cm.config.SK,
	}
	if cm.config.Stop != nil {
		req.ModelConfig.Stop = *cm.config.Stop
	}
	if cm.config.ResponseFormat != nil {
		req.ModelConfig.ResponseFormat = &gateway.ResponseFormat{
			Type: *cm.config.ResponseFormat,
		}
	}

	return req, nil
}
