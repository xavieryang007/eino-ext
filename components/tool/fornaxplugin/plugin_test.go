package fornaxplugin

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/bytedance/mockey"
	"github.com/stretchr/testify/assert"

	"code.byted.org/flow/eino/components/model"
	compTool "code.byted.org/flow/eino/components/tool"
	"code.byted.org/flow/eino/compose"
	"code.byted.org/flow/eino/flow/agent/react"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/kite/kitex/client/callopt"
	"code.byted.org/lang/gg/gptr"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/domain/definition"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/domain/tool"
	"code.byted.org/overpass/flow_devops_plugin/kitex_gen/flow/devops/plugin/executor"
)

func TestIntentionAndExecution(t *testing.T) {
	InitENV("boe_fornax_tool")
	InitCallOpts(callopt.WithRPCTimeout(5 * time.Second))

	defer mockey.Mock(mockey.GetMethod(pluginClient, "MGetToolDescriptor")).Return(
		&executor.MGetToolDescriptorResponse{
			Descriptors: map[int64]*tool.Tool{
				1: {
					Id:       1,
					Name:     "search",
					Desc:     "头条搜索",
					PluginId: 1,
					RequestDefinition: &definition.Definition{
						Type: definition.DataTypeObject,
						Properties: map[string]*definition.Definition{
							"body": {
								Type: definition.DataTypeObject,
								Properties: map[string]*definition.Definition{
									"input_query": {
										Type:        definition.DataTypeString,
										Description: gptr.Of("用户输入的查询的关键字内容"),
									},
								},
								Required: []string{"input_query"},
							},
						},
						Required: []string{"body"},
					},
					ResponseDefinition: nil,
					ServiceType:        tool.ServiceTypePtr(tool.ServiceType_SeedPluginPlatform),
					SeedPluginPlatformToolInfo: &tool.SeedPluginPlatformToolInfo{
						PluginName: "SearchPlugin",
						PluginId:   "47",
						ToolName:   "SearchPlugin",
					},
				},
			},
		}, nil).Build().Patch().UnPatch()

	defer mockey.Mock(mockey.GetMethod(pluginClient, "Execute")).Return(
		&executor.ExecuteResponse{
			Data: &tool.ToolPayload{
				Payload: "{\n  \"results\": [\n    {\n      \"score\": 0.99,\n      \"content\": \"中国夺得100块金牌，50块银牌\"\n    },\n    {\n      \"score\": 0.98,\n      \"content\": \"中国夺得90块金牌，45块银牌\"\n    }\n  ]\n}",
			},
		}, nil).Build().Patch().UnPatch()

	ctx := context.Background()

	chatModel := &mockChatModel{times: 0}

	plg, err := NewPluginTool(ctx, &PluginToolConfig{
		ToolID: 1,
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

func TestNewPluginTool(t *testing.T) {

	InitENV("boe_fornax_tool")
	InitCallOpts(callopt.WithRPCTimeout(5 * time.Second))

	ctx := context.Background()

	defer mockey.Mock(mockey.GetMethod(pluginClient, "MGetToolDescriptor")).Return(
		&executor.MGetToolDescriptorResponse{
			Descriptors: map[int64]*tool.Tool{
				1: {
					Id:       1,
					Name:     "search",
					Desc:     "头条搜索",
					PluginId: 1,
					RequestDefinition: &definition.Definition{
						Type: definition.DataTypeObject,
						Properties: map[string]*definition.Definition{
							"body": {
								Type: definition.DataTypeObject,
								Properties: map[string]*definition.Definition{
									"input_query": {
										Type:        definition.DataTypeString,
										Description: gptr.Of("用户输入的查询的关键字内容"),
									},
								},
								Required: []string{"input_query"},
							},
						},
						Required: []string{"body"},
					},
					ResponseDefinition: nil,
					ServiceType:        tool.ServiceTypePtr(tool.ServiceType_SeedPluginPlatform),
					SeedPluginPlatformToolInfo: &tool.SeedPluginPlatformToolInfo{
						PluginName: "SearchPlugin",
						PluginId:   "47",
						ToolName:   "SearchPlugin",
					},
				},
			},
		}, nil).Build().Patch().UnPatch()

	defer mockey.Mock(mockey.GetMethod(pluginClient, "Execute")).Return(
		&executor.ExecuteResponse{
			Data: &tool.ToolPayload{
				Payload: "{\n  \"results\": [\n    {\n      \"score\": 0.99,\n      \"content\": \"中国夺得100块金牌，50块银牌\"\n    },\n    {\n      \"score\": 0.98,\n      \"content\": \"中国夺得90块金牌，45块银牌\"\n    }\n  ]\n}",
			},
		}, nil).Build().Patch().UnPatch()

	plg, err := NewPluginTool(ctx, &PluginToolConfig{
		ToolID: 1,
	})
	assert.NoError(t, err)

	toolInfo, err := plg.Info(ctx)
	assert.NoError(t, err)

	assert.Equal(t, &schema.ToolInfo{
		Name: "search",
		Desc: "头条搜索",
		ParamsOneOf: schema.NewParamsOneOfByParams(
			map[string]*schema.ParameterInfo{
				"body": {
					Type:     schema.Object,
					Required: true,
					SubParams: map[string]*schema.ParameterInfo{
						"input_query": {
							Type:     schema.String,
							Desc:     "用户输入的查询的关键字内容",
							Required: true,
						},
					},
				},
			}),
	}, toolInfo)

	content, err := plg.InvokableRun(ctx, "{\"body\":{ \"input_query\": \"巴黎奥运会中国队获取了多少金牌\" }}",
		WithKiteCallOptions(callopt.WithRPCTimeout(10*time.Second)))
	assert.NoError(t, err)

	assert.JSONEq(t, "{\n  \"results\": [\n    {\n      \"score\": 0.99,\n      \"content\": \"中国夺得100块金牌，50块银牌\"\n    },\n    {\n      \"score\": 0.98,\n      \"content\": \"中国夺得90块金牌，45块银牌\"\n    }\n  ]\n}",
		content)
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
				Name:      "search",
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
