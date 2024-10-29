package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/getkin/kin-openapi/openapi3gen"

	"code.byted.org/flow/eino/schema"

	"code.byted.org/flow/eino-ext/components/model/openai"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")

	type Person struct {
		Name   string `json:"name"`
		Height int    `json:"height"`
		Weight int    `json:"weight"`
	}
	personSchema, err := openapi3gen.NewSchemaRefForValue(&Person{}, nil)
	if err != nil {
		panic(err)
	}

	ctx := context.Background()
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey: accessKey,
		Model:  "gpt-4o",
		ResponseFormat: &openai.ChatCompletionResponseFormat{
			Type: openai.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai.ChatCompletionResponseFormatJSONSchema{
				Name:        "person",
				Description: "data that describes a person",
				Strict:      false,
				Schema:      personSchema.Value,
			},
		},
	})
	if err != nil {
		panic(fmt.Errorf("NewChatModel failed, err=%v", err))
	}

	resp, err := chatModel.Generate(ctx, []*schema.Message{
		{
			Role:    schema.System,
			Content: "Parse the user input into the specified json struct",
		},
		{
			Role:    schema.User,
			Content: "John is one meter seventy tall and weighs sixty kilograms",
		},
	})

	if err != nil {
		panic(fmt.Errorf("generate failed, err=%v", err))
	}

	result := &Person{}
	err = json.Unmarshal([]byte(resp.Content), result)
	if err != nil {
		panic(fmt.Errorf("unmarshal failed, err=%v", err))
	}
	fmt.Printf("%+v", *result)
}
