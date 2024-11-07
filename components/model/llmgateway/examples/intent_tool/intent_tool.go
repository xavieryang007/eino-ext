package main

import (
	"code.byted.org/flow/eino-ext/components/model/llmgateway"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
	"context"
	"encoding/json"

	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/logs/v2"
)

func main() {
	ctx := context.Background()
	chatModel, err := llmgateway.NewChatModel(ctx, &llmgateway.ChatModelConfig{
		Model: "1727236836", // change to your `model id`
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	err = chatModel.BindTools([]*schema.ToolInfo{
		{
			Name: "user_company",
			Desc: "根据用户的姓名和邮箱，查询用户的公司和职位信息",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"name": {
						Type: "string",
						Desc: "用户的姓名",
					},
					"email": {
						Type: "string",
						Desc: "用户的邮箱",
					},
				}),
		},
		{
			Name: "user_salary",
			Desc: "根据用户的姓名和邮箱，查询用户的薪酬信息",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"name": {
						Type: "string",
						Desc: "用户的姓名",
					},
					"email": {
						Type: "string",
						Desc: "用户的邮箱",
					},
				}),
		},
	})
	if err != nil {
		logs.Errorf("BindForcedTools failed, err=%v", err)
		return
	}

	// required
	info := &gateway.UserInfo{
		AppId:  0,
		ApiKey: "your-api-key",
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一名房产经纪人，结合用户的薪酬和工作，使用 user_company、user_salary 两个 API，为其提供相关的房产信息。邮箱是必须的",
		},
		{
			Role:    schema.User,
			Content: "我的姓名是 zhangsan，我的邮箱是 zhangsan@bytedance.com，请帮我推荐一些适合我的房子。",
		},
	}, llmgateway.WithUserInfo(info))

	if err != nil {
		logs.Errorf("Generate failed, err=%v", err)
		return
	}

	respBytes, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		logs.Errorf("json.MarshalIndent failed, err=%v", err)
		return
	}

	logs.Infof("output: \n%v", string(respBytes))
}
