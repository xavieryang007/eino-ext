package cozeplugin

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/callbacks"
	"code.byted.org/flow/eino/components/model"
	compTool "code.byted.org/flow/eino/components/tool"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/flow/agent/react"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/kite/kitex/pkg/streaming"
	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin"
)

func TestInvokable(t *testing.T) {
	pluginID := int64(1)
	pluginName := "plugin name"
	apiID := int64(2)
	apiName := "api name"
	apiDesc := "api desc"
	var params []*plugin.Parameter
	resp := "hello"

	defer mockey.Mock(mockey.GetMethod(kitexClient, "GetPluginList")).Return(
		&plugin.GetPluginListResponse{
			GetPluginListData: &plugin.GetPluginListData{
				PluginInfos: []*plugin.PluginInfo{
					{
						Id:   pluginID,
						Name: pluginName,
						APIs: []*plugin.API{
							{
								Name:       apiName,
								Desc:       apiDesc,
								Parameters: params,
								PluginId:   pluginID,
								PluginName: pluginName,
								ApiID:      apiID,
							},
						},
					},
				},
			},
		}, nil).Build().Patch().UnPatch()

	defer mockey.Mock(mockey.GetMethod(kitexClient, "DoAction")).Return(
		&plugin.DoActionResponse{
			Resp:    resp,
			Success: true,
		}, nil).Build().Patch().UnPatch()

	ctx := context.Background()

	chatModel := &mockChatModel{times: 0}

	plg, err := NewTool(ctx, &Config{
		PluginID: pluginID,
		APIID:    apiID,
	})
	assert.NoError(t, err)

	msgModifier := react.NewPersonaModifier(`你是一名信息咨询助手，针对用户的问题，进行调用 'search' 工具，并根据 'search' 工具的查询结果进行回答`)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: chatModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []compTool.BaseTool{plg},
		},
		MessageModifier: msgModifier,
	})
	assert.NoError(t, err)

	ah := &mockHandler{}
	out, err := agent.Generate(
		ctx,
		[]*schema.Message{
			schema.UserMessage("中国在巴黎奥运会总过获得了多少奖牌"),
		},
		react.WithCallbacks(ah),
	)
	assert.NoError(t, err)
	assert.Equal(t, "4", out.Content)
	assert.Equal(t, 0, ah.cnt)
	//assert.Equal(t, 2, ah.cnt)
}

func TestStreamable(t *testing.T) {
	pluginID := int64(1)
	pluginName := "plugin name"
	apiID := int64(2)
	apiName := "api name"
	apiDesc := "api desc"
	defer mockey.Mock(mockey.GetMethod(kitexClient, "GetPluginList")).Return(
		&plugin.GetPluginListResponse{
			GetPluginListData: &plugin.GetPluginListData{
				PluginInfos: []*plugin.PluginInfo{
					{
						Id:   pluginID,
						Name: pluginName,
						APIs: []*plugin.API{
							{
								Name:       apiName,
								Desc:       apiDesc,
								PluginId:   pluginID,
								PluginName: pluginName,
								ApiID:      apiID,
								RunMode:    plugin.RunMode_Streaming,
							},
						},
					},
				},
			},
		}, nil).Build().Patch().UnPatch()

	expStreamItems := 2
	str := "asd"
	sc := &mockStreamClient{str: str, maxTimes: expStreamItems}
	defer mockey.Mock(mockey.GetMethod(kitexClient, "StreamDoAction")).Return(sc, nil).Build().UnPatch()

	ctx := context.Background()
	tool, err := NewTool(ctx, &Config{PluginID: pluginID, APIID: apiID})
	assert.NoError(t, err)
	info, err := tool.Info(ctx)
	assert.NoError(t, err)
	assert.Equal(t, apiDesc, info.Desc)
	assert.Equal(t, apiName, info.Name)

	st, ok := tool.(*streamCozePlugin)
	assert.True(t, ok)

	handlerCallTimes := 0
	cbm, ok := callbacks.NewManager(&callbacks.RunInfo{}, &callbacks.HandlerBuilder{
		OnStartFn: func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
			if _, ok := GetCallbackInputExtraDetail(compTool.ConvCallbackInput(input)); ok {
				handlerCallTimes++
			}

			return ctx
		},
		OnEndWithStreamOutputFn: func(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
			for {
				co, err := output.Recv()
				if err == io.EOF {
					break
				}

				if _, ok := GetCallbackOutputExtraDetail(compTool.ConvCallbackOutput(co)); ok {
					handlerCallTimes++
				}
			}

			output.Close()
			return ctx
		},
	})
	assert.True(t, ok)

	sr, err := st.StreamableRun(callbacks.CtxWithManager(ctx, cbm), "hello")
	assert.NoError(t, err)

	srLen := 0
	for {
		s, err := sr.Recv()
		if err == io.EOF {
			break
		}

		srLen++
		assert.Equal(t, str, s)
	}

	sr.Close()
	assert.Equal(t, expStreamItems, srLen)
	assert.Equal(t, 1+expStreamItems, handlerCallTimes) // start + chan items
}

func TestGetCallbackInputExtraDetail(t *testing.T) {
	var d *InputExtraDetail
	var ok bool

	d, ok = GetCallbackInputExtraDetail(nil)
	assert.False(t, ok)
	assert.Nil(t, d)

	d, ok = GetCallbackInputExtraDetail(&compTool.CallbackInput{})
	assert.False(t, ok)
	assert.Nil(t, d)

	d, ok = GetCallbackInputExtraDetail(&compTool.CallbackInput{Extra: map[string]any{callbackExtraInputDetail: 123}})
	assert.False(t, ok)
	assert.Nil(t, d)

	d, ok = GetCallbackInputExtraDetail(&compTool.CallbackInput{Extra: map[string]any{callbackExtraInputDetail: &InputExtraDetail{}}})
	assert.True(t, ok)
	assert.NotNil(t, d)
}

func TestGetCallbackOutputExtraDetail(t *testing.T) {
	var d *OutputExtraDetail
	var ok bool

	d, ok = GetCallbackOutputExtraDetail(nil)
	assert.False(t, ok)
	assert.Nil(t, d)

	d, ok = GetCallbackOutputExtraDetail(&compTool.CallbackOutput{})
	assert.False(t, ok)
	assert.Nil(t, d)

	d, ok = GetCallbackOutputExtraDetail(&compTool.CallbackOutput{Extra: map[string]any{callbackExtraOutputDetail: 123}})
	assert.False(t, ok)
	assert.Nil(t, d)

	d, ok = GetCallbackOutputExtraDetail(&compTool.CallbackOutput{Extra: map[string]any{callbackExtraOutputDetail: &OutputExtraDetail{}}})
	assert.True(t, ok)
	assert.NotNil(t, d)
}

type mockChatModel struct {
	times int
}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	m.times++
	if m.times == 1 {
		return schema.AssistantMessage("", []schema.ToolCall{{
			ID: "123456",
			Function: schema.FunctionCall{
				Name:      "api name",
				Arguments: "{\"body\":{ \"input_query\": \"巴黎奥运会中国队获取了多少金牌\" }}",
			},
		}}), nil
	} else if m.times == 2 {
		return schema.AssistantMessage(fmt.Sprintf("%v", len(input)), nil), nil
	}
	return nil, nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	return nil
}

type mockHandler struct {
	react.BaseCallback
	cnt int
}

func (m *mockHandler) OnToolStart(ctx context.Context, input string) {
	// todo: fix after eino agent callbacks refactor
	//if _, ok := GetCallbackInputExtraDetail(input); ok {
	//	m.cnt++
	//}
}

func (m *mockHandler) OnToolEnd(ctx context.Context, output string) {
	// todo: fix after eino agent callbacks refactor
	//if _, ok := GetCallbackOutputExtraDetail(output); ok {
	//	m.cnt++
	//}
}

func (m *mockHandler) OnToolEndStream(ctx context.Context, output *schema.StreamReader[string]) {
	output.Close()
}

type mockStreamClient struct {
	streaming.Stream

	str       string
	callTimes int
	maxTimes  int
}

func (s *mockStreamClient) Close() error {
	return nil
}

func (s *mockStreamClient) Recv() (*plugin.StreamDoActionResponse, error) {
	if s.callTimes == s.maxTimes {
		return nil, io.EOF
	}

	s.callTimes++
	return &plugin.StreamDoActionResponse{
		Resp: &plugin.StreamDoActionData{
			SSEData: &s.str,
		},
		Success:             true,
		Tokens:              nil,
		Cost:                nil,
		IsFinish:            false,
		SSEEventId:          "123",
		PluginInterruptData: nil,
	}, nil
}
