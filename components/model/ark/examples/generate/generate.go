package main

import (
	"context"
	"log"
	"os"

	"code.byted.org/flow/eino-ext/components/model/ark"
	"code.byted.org/flow/eino/schema"
)

func main() {
	ctx := context.Background()

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL_ID"),
	})
	if err != nil {
		log.Printf("NewChatModel failed, err=%v", err)
		return
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	})
	if err != nil {
		log.Printf("Generate failed, err=%v", err)
		return
	}

	log.Printf("output: \n%v", resp)
}
