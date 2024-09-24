package main

import (
	"context"
	"log"
	"os"

	"code.byted.org/flow/eino-ext/components/embedding/maas"
)

func main() {
	ctx := context.Background()

	embedder, err := maas.NewEmbedder(ctx, &maas.EmbeddingConfig{
		// you can get key from https://cloud.bytedance.net/ark/region:ark+cn-beijing/endpoint
		// attention: model must support embedding, for example: doubao-embedding
		APIKey: os.Getenv("MAAS_API_KEY"), // for example, "xxxxxx-xxxx-xxxx-xxxx-xxxxxxx"
		Model:  os.Getenv("MAAS_MODEL"),   // for example, "ep-20240909094235-xxxx"
	})
	if err != nil {
		log.Printf("new embedder error: %v\n", err)
		return
	}

	embedding, err := embedder.EmbedStrings(ctx, []string{"hello world", "hello world"})
	if err != nil {
		log.Printf("embedding error: %v\n", err)
		return
	}

	log.Printf("embedding: %v\n", embedding)
}
