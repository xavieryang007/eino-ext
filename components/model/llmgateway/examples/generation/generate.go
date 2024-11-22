package main

import (
	"context"
	"encoding/json"
	"os"
	"strconv"

	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"

	"code.byted.org/flow/eino-ext/components/model/llmgateway"
)

func main() {
	id := logid.GenLogID()
	ctx := context.Background()
	ctx = ctxvalues.SetLogID(ctx, id)

	llmGWAppID := os.Getenv("LLM_GW_APP_ID")
	llmGWApiKey := os.Getenv("LLM_GW_API_KEY")
	ak := os.Getenv("API_KEY")
	sk := os.Getenv("SECRET_KEY")

	chatModel, err := llmgateway.NewChatModel(ctx, &llmgateway.ChatModelConfig{
		Model: "1727236836", // change to your `model id`
		AK:    &ak,
		SK:    &sk,
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	appID, err := strconv.Atoi(llmGWAppID)
	if err != nil {
		logs.Errorf("strconv.Atoi(llmGWAppID) failed, err=%v", err)
	}
	// required
	info := &gateway.UserInfo{
		AppId:  int64(appID),
		ApiKey: llmGWApiKey,
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

	msgData, _ := json.Marshal(resp)
	logs.Infof("log_id=%v, output: \n%v", id, string(msgData))
}
