package main

import (
	"context"
	"os"

	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/ark"
)

func main() {
	ctx := context.Background()

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL_ID"),
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

	logs.Infof("output: \n%v", resp)
}
