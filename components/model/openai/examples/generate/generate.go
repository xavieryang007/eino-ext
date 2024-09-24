package main

import (
	"context"
	"os"

	"code.byted.org/flow/eino-ext/components/model/openai"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")

	id := logid.GenLogID()
	ctx := context.Background()
	ctx = ctxvalues.SetLogID(ctx, id)

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://search.bytedance.net/gpt/openapi/online/multimodal/crawl",
		APIKey:  accessKey,
		ByAzure: true,
		Model:   "gpt-4o-2024-05-13",
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	})
	if err != nil {
		logs.Errorf("Generate failed, err=%v", err)
		return
	}

	logs.Infof("log_id=%v, output: \n%v", id, resp)
}
