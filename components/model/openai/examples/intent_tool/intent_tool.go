package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/openai"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")
	ctx := context.Background()
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  accessKey,
		ByAzure: false,
		Model:   "gpt-4o",
	})
	if err != nil {
		panic(fmt.Errorf("NewChatModel failed, err=%v", err))
	}
	err = chatModel.BindForcedTools([]*schema.ToolInfo{
		{
			Name: "user_company",
			Desc: "Retrieve the user's company and position based on their name and email.",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"name":  {Type: "string", Desc: "user's name"},
					"email": {Type: "string", Desc: "user's email"}}),
		}, {
			Name: "user_salary",
			Desc: "Retrieve the user's salary based on their name and email.\n",
			ParamsOneOf: schema.NewParamsOneOfByParams(
				map[string]*schema.ParameterInfo{
					"name":  {Type: "string", Desc: "user's name"},
					"email": {Type: "string", Desc: "user's email"},
				}),
		}})
	if err != nil {
		panic(fmt.Errorf("BindForcedTools failed, err=%v", err))
	}
	resp, err := chatModel.Generate(ctx, []*schema.Message{{
		Role:    schema.System,
		Content: "As a real estate agent, provide relevant property information based on the user's salary and job using the user_company and user_salary APIs. An email address is required.",
	}, {
		Role:    schema.User,
		Content: "My name is John and my email is john@abc.com，Please recommend some houses that suit me.",
	}})
	if err != nil {
		panic(fmt.Errorf("generate failed, err=%v", err))
	}
	fmt.Printf("output: \n%v", resp)

	streamResp, err := chatModel.Stream(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "As a real estate agent, provide relevant property information based on the user's salary and job using the user_company and user_salary APIs. An email address is required.",
		}, {
			Role:    schema.User,
			Content: "My name is John and my email is john@abc.com，Please recommend some houses that suit me.",
		},
	})
	if err != nil {
		panic(fmt.Errorf("generate failed, err=%v", err))
	}
	var messages []*schema.Message
	for {
		chunk, err := streamResp.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(fmt.Errorf("recv failed, err=%v", err))
		}
		messages = append(messages, chunk)
	}
	resp, err = schema.ConcatMessages(messages)
	if err != nil {
		panic(fmt.Errorf("ConcatMessages failed, err=%v", err))
	}
	fmt.Printf("stream output: \n%v", resp)
}
