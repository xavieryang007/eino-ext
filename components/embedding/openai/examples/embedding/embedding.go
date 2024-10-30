package main

import (
	"context"
	"fmt"
	"os"

	"code.byted.org/flow/eino-ext/components/embedding/openai"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")

	ctx := context.Background()

	var (
		defaultDim = 1024
	)

	embedding, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		APIKey:     accessKey,
		Model:      "text-embedding-3-large",
		Dimensions: &defaultDim,
		Timeout:    0,
	})
	if err != nil {
		panic(fmt.Errorf("new embedder error: %v\n", err))
	}

	resp, err := embedding.EmbedStrings(ctx, []string{"hello", "how are you"})
	if err != nil {
		panic(fmt.Errorf("generate failed, err=%v", err))
	}

	fmt.Printf("output=%v", resp)
}
