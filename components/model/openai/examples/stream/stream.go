package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/openai"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")

	N := 3
	ctx := context.Background()
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: "https://search.bytedance.net/gpt/openapi/online/multimodal/crawl",
		N:       &N,
		APIKey:  accessKey,
		ByAzure: true,
		Model:   "gpt-4o-2024-05-13",
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	streamMsgs, err := chatModel.Stream(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	})

	if err != nil {
		logs.Errorf("Generate failed, err=%v", err)
		return
	}

	defer streamMsgs.Close()

	logs.Infof("typewriter output:")
	for {
		msg, err := streamMsgs.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("\nstream.Recv failed, err=%v", err)
			return
		}
		fmt.Print(msg.Content)
	}

	fmt.Print("\n")
}
