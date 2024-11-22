package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/kite/kitutil"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"

	"code.byted.org/flow/eino-ext/components/model/llmgateway"
)

func main() {
	ctx := context.Background()

	llmGWAppID := os.Getenv("LLM_GW_APP_ID")
	llmGWApiKey := os.Getenv("LLM_GW_API_KEY")
	ak := os.Getenv("API_KEY")
	sk := os.Getenv("SECRET_KEY")

	// Bind your own env
	ctx = kitutil.NewCtxWithEnv(ctx, "boe_llm_gateway")

	var t float32 = 0.7
	chatModel, err := llmgateway.NewChatModel(ctx, &llmgateway.ChatModelConfig{
		Model:       "1727236836", // change to your `model id`
		AK:          &ak,
		SK:          &sk,
		Temperature: &t,
	})

	appID, err := strconv.Atoi(llmGWAppID)
	if err != nil {
		logs.Errorf("strconv.Atoi(llmGWAppID) failed, err=%v", err)
	}
	// required
	info := &gateway.UserInfo{
		AppId:  int64(appID),
		ApiKey: llmGWApiKey,
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
		// fmt.Print(msg.Content)
		// if raw, ok := llmgateway.GetRawResp(msg); ok {
		// 	logs.Infof("raw: %v", raw)
		// }

		msgData, _ := json.Marshal(msg)
		fmt.Printf("eino chunk message: %#v\n\n", string(msgData))
	}

	fmt.Print("\n")
}
