package main

import (
	"context"
	"os"

	"code.byted.org/flow/eino-ext/components/embedding/openai"
	"code.byted.org/gopkg/logs/v2"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")

	ctx := context.Background()

	embedding, err := openai.NewEmbedder(ctx, &openai.EmbeddingConfig{
		BaseURL:        "https://search.bytedance.net/gpt/openapi/online/multimodal/crawl",
		APIKey:         accessKey,
		ByAzure:        true,
		Model:          "text-embedding-3-large",
		EncodingFormat: openai.EmbeddingEncodingFormatFloat,
		Dimensions:     1024,
		Timeout:        0,
	})
	if err != nil {
		logs.Errorf("new embedder error: %v\n", err)
		return
	}

	resp, err := embedding.EmbedStrings(ctx, []string{"hello", "how are you"})
	if err != nil {
		logs.Errorf("Generate failed, err=%v", err)
		return
	}

	logs.CtxInfo(ctx, "output=%v", resp)
}
