package main

import (
	"context"
	"fmt"
	"os"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/openai"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")

	ctx := context.Background()

	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		// if you want to use Azure OpenAI Service, set these two field.
		// BaseURL: "https://{RESOURCE_NAME}.openai.azure.com",
		// ByAzure: true,
		// APIVersion: "2024-06-01",
		APIKey: accessKey,
		Model:  "gpt-4o-2024-05-13",
	})
	if err != nil {
		panic(fmt.Errorf("NewChatModel failed, err=%v", err))
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	})
	if err != nil {
		panic(fmt.Errorf("generate failed, err=%v", err))
	}

	fmt.Printf("output: \n%v", resp)
}
