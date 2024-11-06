package main

import (
	"context"

	"code.byted.org/flow/eino-ext/components/model/ollama"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"
)

func main() {
	id := logid.GenLogID()
	ctx := context.Background()
	ctx = ctxvalues.SetLogID(ctx, id)

	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: "http://localhost:11434",
		Model:   "llama3",
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.User,
			Content: "as a machine, how do you answer user's question?",
		},
	})
	if err != nil {
		logs.Errorf("Generate failed, err=%v", err)
		return
	}

	logs.Infof("log_id=%v, output: \n%v", id, resp)
}
