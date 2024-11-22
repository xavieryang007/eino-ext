package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/ark"
)

func main() {
	ctx := context.Background()

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL_ID"),
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

	defer streamMsgs.Close() // do not forget to close the stream

	msgs := make([]*schema.Message, 0)

	logs.Infof("typewriter output:")
	for {
		msg, err := streamMsgs.Recv()
		if err == io.EOF {
			break
		}
		msgs = append(msgs, msg)
		if err != nil {
			logs.Errorf("\nstream.Recv failed, err=%v", err)
			return
		}
		fmt.Print(msg.Content)
	}

	msg, err := schema.ConcatMessages(msgs)
	if err != nil {
		logs.Errorf("ConcatMessages failed, err=%v", err)
		return
	}

	logs.Infof("output: %s\n", msg.Content)
}
