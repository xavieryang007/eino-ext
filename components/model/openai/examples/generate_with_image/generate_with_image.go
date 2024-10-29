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
		APIKey:  accessKey,
		ByAzure: false,
		Model:   "gpt-4o-2024-05-13",
	})
	if err != nil {
		panic(fmt.Errorf("NewChatModel failed, err=%v", err))

	}

	multiModalMsg := schema.UserMessage("")
	multiModalMsg.MultiContent = []schema.ChatMessagePart{
		{
			Type: schema.ChatMessagePartTypeText,
			Text: "this picture is a landscape photo, what's the picture's content",
		},
		{
			Type: schema.ChatMessagePartTypeImageURL,
			ImageURL: &schema.ChatMessageImageURL{
				URL:    "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcT11qEDxU4X_MVKYQVU5qiAVFidA58f8GG0bQ&s",
				Detail: schema.ImageURLDetailAuto,
			},
		},
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		multiModalMsg,
	})
	if err != nil {
		panic(fmt.Errorf("generate failed, err=%v", err))
	}

	fmt.Printf("output: \n%v", resp)
}
