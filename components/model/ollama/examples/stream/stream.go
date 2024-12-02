package main

import (
	"context"
	"fmt"
	"io"
	"log"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/ollama"
)

func main() {
	ctx := context.Background()
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: "http://localhost:11434",
		Model:   "llama3",
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

	defer streamMsgs.Close()

	log.Println("typewriter output:")
	for {
		msg, err := streamMsgs.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("\nstream.Recv failed, err=%v", err)
			return
		}
		fmt.Print(msg.Content)
	}

	fmt.Print("\n")
}
