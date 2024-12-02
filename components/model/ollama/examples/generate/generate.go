package main

import (
	"context"
	"log"

	"code.byted.org/flow/eino-ext/components/model/ollama"
	"code.byted.org/flow/eino/schema"
)

func main() {
	ctx := context.Background()

	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: "http://localhost:11434",
		Model:   "llama3",
	})
	if err != nil {
		log.Printf("NewChatModel failed, err=%v\n", err)
		return
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	})
	if err != nil {
		log.Printf("Generate failed, err=%v\n", err)
		return
	}

	log.Printf("output: \n%v\n", resp)
}
