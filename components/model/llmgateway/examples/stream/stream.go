package main

import (
	"code.byted.org/flow/eino-ext/components/model/llmgateway"
	"code.byted.org/kite/kitutil"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
	"context"
	"fmt"
	"io"

	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino/schema"
)

func main() {
	ctx := context.Background()
	// Bind your own env
	ctx = kitutil.NewCtxWithEnv(ctx, "boe_llm_gateway")

	var t float32 = 0.7
	chatModel, err := llmgateway.NewChatModel(ctx, &llmgateway.ChatModelConfig{
		Model: "38", // change to your `model id`
		//AK:          conv.StringPtr("your-ak"),
		//SK:          conv.StringPtr("your-sk"),
		Temperature: &t,
	})

	// required
	info := &gateway.UserInfo{
		AppId:  0,
		ApiKey: "your-api-key",
	}

	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	message := &schema.Message{
		Role:    schema.User,
		Content: "as a machine, how do you answer user's question?",
		MultiContent: []schema.ChatMessagePart{
			{
				Type: schema.ChatMessagePartTypeText,
				Text: fmt.Sprintf("hello world"),
			},
			{
				Type: schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{
					URL: "image.Url",
				},
			},
			{
				Type: schema.ChatMessagePartTypeFileURL,
				FileURL: &schema.ChatMessageFileURL{
					URL:  "file.Url",
					URI:  "file.Uri",
					Name: "file.Name",
				},
			},
		},
	}

	streamMsgs, err := chatModel.Stream(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "你是一个搞笑机器人，每次说话都会先说哈哈哈",
		},
		message,
	}, llmgateway.WithUserInfo(info))

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
		if raw, ok := llmgateway.GetRawResp(msg); ok {
			logs.Infof("raw: %v", raw)
		}
	}

	fmt.Print("\n")
}
