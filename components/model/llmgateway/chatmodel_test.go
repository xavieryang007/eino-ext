package llmgateway

import (
	"context"
	"fmt"
	"io"
	"testing"

	. "github.com/bytedance/mockey"
	"github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/lang/conv"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"

	"code.byted.org/flow/eino-ext/components/model/llmgateway/internal/mock/llmgatewayservice"
)

func mockStreamMessage(cli *llmgatewayservice.MockLLMGatewayService_ChatClient) {
	cli.EXPECT().Close().Return(nil)
	cli.EXPECT().Recv().Return(&gateway.ChatCompletion{
		Choices: []*gateway.Choice{
			{
				Delta: &gateway.Message{
					Role:    "assistant",
					Content: "hello ",
				},
			},
		},
	}, nil).Times(1)
	cli.EXPECT().Recv().Return(&gateway.ChatCompletion{
		Choices: []*gateway.Choice{
			{
				Delta: &gateway.Message{
					Role:    "assistant",
					Content: "world",
				},
			},
		},
		Usage: &gateway.Usage{
			PromptTokens:     1,
			CompletionTokens: 1,
			TotalTokens:      2,
		},
	}, nil).Times(1)
	cli.EXPECT().Recv().Return(nil, io.EOF).Times(1)
}

func mockToolMessage(cli *llmgatewayservice.MockLLMGatewayService_ChatClient) {
	cli.EXPECT().Close().Return(nil)
	cli.EXPECT().Recv().Return(&gateway.ChatCompletion{
		Choices: []*gateway.Choice{
			{
				Delta: &gateway.Message{
					Role: "assistant",
					ToolCalls: []*gateway.ToolCall{
						{
							Index: 0,
							Id:    "call_id_1",
							Function_: &gateway.FunctionCall{
								Name:      "get_current_weather",
								Arguments: `{"location": "Boston, MA", "format": "celsius"}`,
							},
						},
					},
				},
			},
		},
	}, nil).Times(1)
	cli.EXPECT().Recv().Return(&gateway.ChatCompletion{
		Choices: []*gateway.Choice{
			{
				Delta: &gateway.Message{
					Role: "assistant",
					ToolCalls: []*gateway.ToolCall{
						{
							Index: 2,
							Id:    "call_id_2",
							Function_: &gateway.FunctionCall{
								Name:      "get_current_weather",
								Arguments: `{"location": "San Francisco", "format": "celsius"}`,
							},
						},
					},
				},
			},
		},
		Usage: &gateway.Usage{
			PromptTokens:     1,
			CompletionTokens: 1,
			TotalTokens:      2,
		},
	}, nil).Times(1)
	cli.EXPECT().Recv().Return(nil, io.EOF).Times(1)
}

func getToolForTest() []*schema.ToolInfo {
	return []*schema.ToolInfo{
		{
			Name: "test",
			Desc: "test",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"test_str": {
						Desc: "test",
						Type: schema.String,
					},
					"test_obj": {
						Desc: "test",
						Type: schema.Object,
						SubParams: map[string]*schema.ParameterInfo{
							"test_int": {
								Desc: "test",
								Type: schema.String,
								Enum: []string{"1", "2"},
							},
						},
					},
					"test_array": {
						Desc: "test",
						Type: schema.Array,
						ElemInfo: &schema.ParameterInfo{
							Desc: "test",
							Type: schema.String,
						},
					},
				}),
		},
	}
}

func TestChatModel(t *testing.T) {
	PatchConvey("test chat generate", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := llmgatewayservice.NewMockLLMGatewayService_ChatClient(ctrl)
		mockStreamMessage(mockService)

		config := &ChatModelConfig{
			Model:       "22",
			MaxTokens:   conv.IntPtr(1024),
			Temperature: conv.Float32Ptr(0.5),
			TopP:        conv.Float32Ptr(3),
		}

		model, err := NewChatModel(ctx, config)
		if err != nil {
			t.Fatalf("NewChatModel failed: %v", err)
		}

		mockCli := model.client

		defer Mock(GetMethod(mockCli, "Chat")).Return(mockService, nil).Build().UnPatch()

		message := &schema.Message{
			Role:    schema.User,
			Content: "hi",
		}
		SetMessageID(message, "mid_1")
		SetExtra(message, map[string]string{"m_key": "m_val"})

		res, err := model.Generate(ctx, []*schema.Message{message},
			WithExtra(map[string]string{"key": "value"}), WithMetaId(12324))
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		convey.So(res.Role, convey.ShouldEqual, schema.Assistant)
		convey.So(res.Content, convey.ShouldEqual, "hello world")
	})

	PatchConvey("test chat stream", t, func() {
		ctx := context.Background()
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := llmgatewayservice.NewMockLLMGatewayService_ChatClient(ctrl)
		mockStreamMessage(mockService)

		config := &ChatModelConfig{
			Model:     "22",
			MaxTokens: conv.IntPtr(1024),
		}

		model, err := NewChatModel(ctx, config)
		if err != nil {
			t.Fatalf("NewChatModel failed: %v", err)
		}

		mockCli := model.client

		defer Mock(GetMethod(mockCli, "Chat")).Return(mockService, nil).Build().UnPatch()

		res, err := model.Stream(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}})
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		msgs := make([]*schema.Message, 0)

		for {
			msg, err := res.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
				t.Fatalf("Recv failed: %v", err)
			}

			msgs = append(msgs, msg)
		}

		convey.So(len(msgs), convey.ShouldEqual, 2)

		finalMsg, err := schema.ConcatMessages(msgs)
		convey.So(err, convey.ShouldBeNil)
		convey.So(finalMsg.Content, convey.ShouldEqual, "hello world")
	})

	convey.Convey("test chat tool calls", t, func() {
		ctx := context.Background()
		cbm, ok := callbacks.NewManager(nil, &cbForTest{})
		if !ok {
			t.Fatalf("callback init err")
		}
		ctx = callbacks.CtxWithManager(ctx, cbm)

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockService := llmgatewayservice.NewMockLLMGatewayService_ChatClient(ctrl)
		mockToolMessage(mockService)

		config := &ChatModelConfig{
			Model:       "22",
			MaxTokens:   conv.IntPtr(1024),
			Temperature: conv.Float32Ptr(0.5),
			TopP:        conv.Float32Ptr(3),
		}

		model, err := NewChatModel(ctx, config)
		if err != nil {
			t.Fatalf("NewChatModel failed: %v", err)
		}

		err = model.BindTools(getToolForTest())
		if err != nil {
			t.Fatalf("BindTools failed: %v", err)
		}

		mockCli := model.client

		defer Mock(GetMethod(mockCli, "Chat")).Return(mockService, nil).Build().UnPatch()

		res, err := model.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "hi"}},
			WithExtra(map[string]string{"key": "value"}),
		)
		if err != nil {
			t.Fatalf("Generate failed: %v", err)
		}

		convey.So(res.Role, convey.ShouldEqual, schema.Assistant)
		convey.So(len(res.ToolCalls), convey.ShouldEqual, 2)
		convey.So(res.ToolCalls[0].Function.Name, convey.ShouldEqual, "get_current_weather")
	})

}

type cbForTest struct {
	callbacks.HandlerBuilder
}

func (cb *cbForTest) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	fmt.Printf("[callback] OnStart")
	return ctx
}
