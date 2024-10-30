package main

import (
	"context"
	"os"

	"code.byted.org/gopkg/logs/v2"

	"code.byted.org/flow/eino-ext/components/embedding/bytedgpt"
)

func main() {
	accessKey := os.Getenv("OPENAI_API_KEY")

	ctx := context.Background()

	var (
		defaultDim = 1024
	)

	embedding, err := bytedgpt.NewEmbedder(ctx, &bytedgpt.EmbeddingConfig{
		BaseURL:    "https://search.bytedance.net/gpt/openapi/online/multimodal/crawl",
		APIKey:     accessKey,
		ByAzure:    true,
		Model:      "text-embedding-3-large",
		Dimensions: &defaultDim,
		Timeout:    0,
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
