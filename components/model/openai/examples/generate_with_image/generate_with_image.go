package main

import (
	"context"
	"os"

	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/openai"
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
		logs.Errorf("Generate failed, err=%v", err)
		return
	}

	logs.Infof("log_id=%v, output: \n%v", id, resp)
}
