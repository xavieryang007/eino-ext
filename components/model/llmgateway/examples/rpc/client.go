package main

import (
	"code.byted.org/gopkg/logs/v2"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway"
	"code.byted.org/overpass/stone_llm_gateway/kitex_gen/stone/llm/gateway/llmgatewayservice"
	"context"
	"io"
)

func main() {
	req := gateway.NewChatRequest()
	req.ModelId = "1727236836" // change to your `model id`
	// required
	req.UserInfo = &gateway.UserInfo{
		AppId:  0,
		ApiKey: "your-api-key",
	}
	req.Arguments = gateway.NewArguments()
	req.ModelConfig = gateway.NewModelConfig()
	req.Messages = append(req.Messages, &gateway.Message{
		Role:    "user",
		Content: "hello",
	})
	streamClient := llmgatewayservice.MustNewStreamClient("stone.llm.gateway")
	ctx := context.Background()
	stream, err := streamClient.Chat(ctx, req)
	if err != nil {
		logs.Errorf("Generate failed, err=%v", err)
	}
	for {
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			logs.Infof("tream is closed")
			return
		} else if recvErr != nil {
			logs.Errorf("\nstream.Recv failed, err=%v", recvErr)
			return
		}
		logs.Infof("raw: %v", msg)
	}
}
