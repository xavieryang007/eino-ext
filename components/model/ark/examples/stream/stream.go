package main

import (
	"context"
	"fmt"
	"io"
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

	streamMsgs, err := chatModel.Stream(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	})

	if err != nil {
		log.Printf("Generate failed, err=%v", err)
		return
	}

	defer streamMsgs.Close() // do not forget to close the stream

	msgs := make([]*schema.Message, 0)

	log.Printf("typewriter output:")
	for {
		msg, err := streamMsgs.Recv()
		if err == io.EOF {
			break
		}
		msgs = append(msgs, msg)
		if err != nil {
			log.Printf("\nstream.Recv failed, err=%v", err)
			return
		}
		fmt.Print(msg.Content)
	}

	msg, err := schema.ConcatMessages(msgs)
	if err != nil {
		log.Printf("ConcatMessages failed, err=%v", err)
		return
	}

	log.Printf("output: %s\n", msg.Content)
}
