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

	// Ark 平台上支持的豆包模型，均不支持多模态能力，暂未测试 Ark 的多模态能力
	multiModalMsg := schema.UserMessage("")
	multiModalMsg.MultiContent = []schema.ChatMessagePart{
		{
			Type: schema.ChatMessagePartTypeText,
			Text: "this picture is LangDChain's architecture, what's the picture's content",
		},
		{
			Type: schema.ChatMessagePartTypeImageURL,
			ImageURL: &schema.ChatMessageImageURL{
				URL:    "https://d2908q01vomqb2.cloudfront.net/887309d048beef83ad3eabf2a79a64a389ab1c9f/2023/07/13/DBBLOG-3334-image001.png",
				Detail: schema.ImageURLDetailAuto,
			},
		},
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		multiModalMsg,
	})
	if err != nil {
		log.Printf("Generate failed, err=%v", err)
		return
	}

	log.Printf("Ark ChatModel output: \n%v", resp)
}
