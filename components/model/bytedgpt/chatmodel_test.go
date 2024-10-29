package bytedgpt

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	goopenai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/flowdevops/mockoai"
)

func mockChatModel(returnTool bool) (url string, close func()) {
	server := mockoai.NewServer()
	if returnTool {
		server.Mock(mockoai.MockConfig{}, func(ctx context.Context, req *mockoai.ChatRequest) ([]mockoai.Choice, error) {
			res := make([]mockoai.Choice, 0)
			res = append(res, mockoai.Choice{
				Message: mockoai.Message{
					Role:       string(schema.Assistant),
					Content:    "hello max",
					ToolCallID: randStr(),
					ToolCalls: []goopenai.ToolCall{
						{
							Type: "function",
							Function: goopenai.FunctionCall{
								Name:      req.Tools[0].Function.Name,
								Arguments: fmt.Sprintf(`{"name": "%s", "hh": "123"}`, randStr()),
							},
						},
					},
				},
			})
			return res, nil
		})
	} else {
		server.Mock(mockoai.MockConfig{}, func(ctx context.Context, req *mockoai.ChatRequest) ([]mockoai.Choice, error) {
			res := make([]mockoai.Choice, 0)
			res = append(res, mockoai.Choice{
				Message: mockoai.Message{
					Role:    string(schema.Assistant),
					Content: "hello " + randStr(),
				},
			})
			return res, nil
		})
	}

	server.Run()

	return server.GetHost(), server.Close
}

func TestChatModelGenerate(t *testing.T) {
	run := func(t *testing.T, ctx context.Context) {
		url, close := mockChatModel(false)
		defer close()

		chatModel, err := NewChatModel(ctx, &ChatModelConfig{ByAzure: true, BaseURL: url, Model: "gpt-3.5-turbo"})
		assert.NoError(t, err)

		msg, err := chatModel.Generate(ctx, []*schema.Message{schema.UserMessage("how are you")})
		assert.Nil(t, err)

		t.Log(msg)
	}
	t.Run("chat model generate", func(t *testing.T) {
		run(t, context.Background())
	})

	t.Run("chat model generate with callback manager", func(t *testing.T) {
		ctx := context.Background()
		cbm, ok := callbacks.NewManager(nil, &callbacks.HandlerBuilder{})
		assert.True(t, ok)
		ctx = callbacks.CtxWithManager(ctx, cbm)

		run(t, ctx)
	})
}

func TestToXXXUtils(t *testing.T) {
	t.Run("toOpenAIMultiContent", func(t *testing.T) {

		multiContents := []schema.ChatMessagePart{
			{
				Type: schema.ChatMessagePartTypeText,
				Text: "image_desc",
			},
			{
				Type: schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{
					URL:    "https://{RL_ADDRESS}",
					Detail: schema.ImageURLDetailAuto,
				},
			},
		}

		mc, err := toOpenAIMultiContent(multiContents)
		assert.NoError(t, err)
		assert.Len(t, mc, 2)
		assert.Equal(t, mc[0], goopenai.ChatMessagePart{
			Type: goopenai.ChatMessagePartTypeText,
			Text: "image_desc",
		})

		assert.Equal(t, mc[1], goopenai.ChatMessagePart{
			Type: goopenai.ChatMessagePartTypeImageURL,
			ImageURL: &goopenai.ChatMessageImageURL{
				URL:    "https://{RL_ADDRESS}",
				Detail: goopenai.ImageURLDetailAuto,
			},
		})

		mc, err = toOpenAIMultiContent(nil)
		assert.Nil(t, err)
		assert.Nil(t, mc)
	})
}

func TestChatModelStream(t *testing.T) {
	run := func(t *testing.T, ctx context.Context) {
		url, close := mockChatModel(false)
		defer close()

		chatModel, err := NewChatModel(ctx, &ChatModelConfig{ByAzure: true, BaseURL: url, Model: "gpt-3.5-turbo"})
		assert.NoError(t, err)

		s, err := chatModel.Stream(ctx, []*schema.Message{schema.UserMessage("how are you")})
		assert.NoError(t, err)

		defer s.Close()

		var m []*schema.Message
		for {
			msg, e := s.Recv()
			if e != nil {
				if e == io.EOF {
					break
				}
				assert.Nil(t, e)
			}

			m = append(m, msg)
		}

		merged, err := schema.ConcatMessages(m)
		assert.Nil(t, err)

		assert.Greater(t, len(m), 1)
		t.Log(merged)
	}

	t.Run("chat model stream", func(t *testing.T) {
		run(t, context.Background())
	})
	t.Run("chat model stream with callback manager", func(t *testing.T) {
		ctx := context.Background()
		cbm, ok := callbacks.NewManager(nil, &callbacks.HandlerBuilder{})
		assert.True(t, ok)
		ctx = callbacks.CtxWithManager(ctx, cbm)
		run(t, ctx)
	})
}

func TestChatModelToolCall(t *testing.T) {

	ctx := context.Background()

	url, close := mockChatModel(true)
	defer close()

	chatModel, err := NewChatModel(ctx, &ChatModelConfig{ByAzure: true, BaseURL: url, Model: "gpt-3.5-turbo"})
	assert.NoError(t, err)

	weatherParams := map[string]*schema.ParameterInfo{
		"location": {
			Type:     schema.String,
			Desc:     "The city and state, e.g. San Francisco, CA",
			Required: true,
		},
		"unit": {
			Type: schema.String,
			Enum: []string{"celsius", "fahrenheit"},
		},
		"days": {
			Type: schema.Array,
			Desc: "The number of days to forecast",
			ElemInfo: &schema.ParameterInfo{
				Type: schema.Integer,
				Desc: "The number of days to forecast",
				Enum: []string{"1", "2", "3", "4", "5", "6", "7"},
			},
		},
		"infos": {
			Type: schema.Object,
			SubParams: map[string]*schema.ParameterInfo{
				"type_windy": {
					Type: schema.Boolean,
					Desc: "The types of windy weather",
				},
				"type_rainy": {
					Type: schema.Boolean,
					Desc: "The types of rainy weather",
				},
			},
		},
	}

	stockParams := map[string]*schema.ParameterInfo{
		"name": {
			Type:     schema.String,
			Desc:     "The name of the stock",
			Required: true,
		},
	}

	weatherToolName := "get_current_weather"
	weatherToolDesc := "Get the current weather in a given location"

	tools := []*schema.ToolInfo{
		{
			Name:        weatherToolName,
			Desc:        weatherToolDesc,
			ParamsOneOf: schema.NewParamsOneOfByParams(weatherParams),
		},
		{
			Name:        "get_current_stock_price",
			Desc:        "Get the current stock price given the name of the stock",
			ParamsOneOf: schema.NewParamsOneOfByParams(stockParams),
		},
	}

	err = chatModel.BindTools(tools)
	assert.Nil(t, err)

	// assert tools
	assert.Len(t, chatModel.tools, 2)
	tool1 := chatModel.tools[0]
	assert.Equal(t, weatherToolName, tool1.Function.Name)
	assert.Equal(t, weatherToolDesc, tool1.Function.Description)
	expectedDefinition := &openapi3.Schema{
		Type: openapi3.TypeObject,
		Properties: map[string]*openapi3.SchemaRef{
			"location": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeString,
					Description: "The city and state, e.g. San Francisco, CA",
				},
			},
			"unit": {
				Value: &openapi3.Schema{
					Type: openapi3.TypeString,
					Enum: []any{"celsius", "fahrenheit"},
				},
			},
			"days": {
				Value: &openapi3.Schema{
					Type:        openapi3.TypeArray,
					Description: "The number of days to forecast",
					Items: &openapi3.SchemaRef{
						Value: &openapi3.Schema{
							Type:        openapi3.TypeInteger,
							Description: "The number of days to forecast",
							Enum:        []any{"1", "2", "3", "4", "5", "6", "7"},
						},
					},
				},
			},
			"infos": {
				Value: &openapi3.Schema{
					Type: openapi3.TypeObject,
					Properties: map[string]*openapi3.SchemaRef{
						"type_windy": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeBoolean,
								Description: "The types of windy weather",
							},
						},
						"type_rainy": {
							Value: &openapi3.Schema{
								Type:        openapi3.TypeBoolean,
								Description: "The types of rainy weather",
							},
						},
					},
					Required: []string{},
				},
			},
		},
		Required: []string{"location"},
	}

	assert.EqualValues(t, tool1.Function.Parameters, expectedDefinition)

	msg, err := chatModel.Generate(ctx,
		[]*schema.Message{schema.UserMessage("what's the weather in Beijing today")})

	assert.Nil(t, err)

	// len(msg.ToolCalls) != 0
	assert.NotZero(t, len(msg.ToolCalls))

	t.Log(msg)

	// stream
	s, err := chatModel.Stream(ctx, []*schema.Message{schema.UserMessage("what's the weather in Beijing today")})
	assert.Nil(t, err)

	defer s.Close()

	var m []*schema.Message
	for {
		msg_, e := s.Recv()
		if e != nil {
			if e == io.EOF {
				break
			}
			assert.Nil(t, e)
		}

		m = append(m, msg_)
	}

	merged, err := schema.ConcatMessages(m)
	assert.Nil(t, err)
	t.Log(merged)
}

func TestChatModelForceToolCall(t *testing.T) {

	t.Run("chat model force tool call", func(t *testing.T) {
		ctx := context.Background()

		url, close := mockChatModel(true)
		defer close()

		chatModel, err := NewChatModel(ctx, &ChatModelConfig{ByAzure: true, BaseURL: url, Model: "gpt-3.5-turbo"})
		assert.NoError(t, err)

		doNothingParams := map[string]*schema.ParameterInfo{
			"test": {
				Type:     schema.String,
				Desc:     "no meaning",
				Required: true,
			},
		}

		stockParams := map[string]*schema.ParameterInfo{
			"name": {
				Type:     schema.String,
				Desc:     "The name of the stock",
				Required: true,
			},
		}

		tools := []*schema.ToolInfo{
			{
				Name:        "do_nothing",
				Desc:        "do nothing",
				ParamsOneOf: schema.NewParamsOneOfByParams(doNothingParams),
			},
			{
				Name:        "get_current_stock_price",
				Desc:        "Get the current stock price given the name of the stock",
				ParamsOneOf: schema.NewParamsOneOfByParams(stockParams),
			},
		}

		err = chatModel.BindForcedTools([]*schema.ToolInfo{tools[0]})
		assert.Nil(t, err)

		msg, err := chatModel.Generate(ctx,
			[]*schema.Message{schema.UserMessage("do not try to call any tool")})

		t.Log(msg)

		assert.Nil(t, err)

		assert.Equal(t, 1, len(msg.ToolCalls))
	})
}

func randStr() string {
	seeds := []rune("abcdefghijklmnopqrstuvwxyz")
	b := make([]rune, 8)
	for i := range b {
		b[i] = seeds[rand.Intn(len(seeds))]
	}
	return string(b)
}

func TestToOpenAIToolCalls(t *testing.T) {
	t.Run("empty tools", func(t *testing.T) {
		tools := toOpenAIToolCalls([]schema.ToolCall{})
		assert.Len(t, tools, 0)
	})

	t.Run("normal tools", func(t *testing.T) {
		fakeToolCall1 := schema.ToolCall{
			ID:       randStr(),
			Function: schema.FunctionCall{Name: randStr(), Arguments: randStr()},
		}

		toolCalls := toOpenAIToolCalls([]schema.ToolCall{fakeToolCall1})

		assert.Len(t, toolCalls, 1)
		assert.Equal(t, fakeToolCall1.ID, toolCalls[0].ID)
		assert.Equal(t, fakeToolCall1.Function.Name, toolCalls[0].Function.Name)
	})
}

func TestRoleTransfer(t *testing.T) {
	tests := []struct {
		name string
		msg  *schema.Message
		want string
	}{
		{
			name: "user",
			msg:  schema.UserMessage("hello"),
			want: goopenai.ChatMessageRoleUser,
		},
		{
			name: "assistant",
			msg:  schema.AssistantMessage("hello", []schema.ToolCall{}),
			want: goopenai.ChatMessageRoleAssistant,
		},
		{
			name: "system",
			msg:  schema.SystemMessage("hello"),
			want: goopenai.ChatMessageRoleSystem,
		},
		{
			name: "tool",
			msg:  schema.ToolMessage("hello", "xxx"),
			want: goopenai.ChatMessageRoleTool,
		},
		{
			name: "user001",
			msg: &schema.Message{
				Role: "user001",
			},
			want: "user001",
		},
	}

	for _, tt := range tests {
		res := string(tt.msg.Role)
		assert.Equal(t, tt.want, res)
	}
}
