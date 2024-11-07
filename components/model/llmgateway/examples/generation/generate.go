package main

import (
	"code.byted.org/flow/eino-ext/components/model/llmgateway"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
	"context"

	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino/schema"
)

func main() {
	id := logid.GenLogID()
	ctx := context.Background()
	ctx = ctxvalues.SetLogID(ctx, id)

	chatModel, err := llmgateway.NewChatModel(ctx, &llmgateway.ChatModelConfig{
		Model: "1727236836", // change to your `model id`
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	// required
	info := &gateway.UserInfo{
		AppId:  0,
		ApiKey: "your-api-key",
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	}, llmgateway.WithUserInfo(info))
	if err != nil {
		logs.Errorf("Generate failed, err=%v", err)
		return
	}

	logs.Infof("log_id=%v, output: \n%v", id, resp)
}
