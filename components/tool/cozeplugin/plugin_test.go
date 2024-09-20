package cozeplugin

import (
	"context"
	"fmt"
	"testing"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"

	"code.byted.org/overpass/ocean_cloud_plugin/kitex_gen/ocean/cloud/plugin"

	"code.byted.org/flow/eino/components/model"
	compTool "code.byted.org/flow/eino/components/tool"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/flow/agent/react"
	"code.byted.org/flow/eino/schema"
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

	out, err := agent.Generate(ctx,
		[]*schema.Message{
			schema.UserMessage("中国在巴黎奥运会总过获得了多少奖牌"),
		},
	)
	assert.NoError(t, err)
	assert.Equal(t, "4", out.Content)
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

	ctx := context.Background()
	tool, err := NewTool(ctx, &Config{PluginID: pluginID, APIID: apiID})
	assert.NoError(t, err)
	info, err := tool.Info(ctx)
	assert.NoError(t, err)
	assert.Equal(t, apiDesc, info.Desc)
	assert.Equal(t, apiName, info.Name)
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
