package main

import (
	"context"
	"os"

	"code.byted.org/flow/eino-ext/components/indexer/vikingdb"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"
	viking "code.byted.org/lagrange/viking_go_client"
)

func main() {
	vikingDBName := os.Getenv("VIKING_DB_NAME")
	vikingDBToken := os.Getenv("VIKING_DB_TOKEN")

	ctx := ctxvalues.SetLogID(context.Background(), logid.GenLogID())

	emb := &mockEmbedding{} // replace with your embedding

	cli, err := vikingdb.NewIndexer(ctx, &vikingdb.IndexerConfig{
		Name:            vikingDBName,
		Token:           vikingDBToken,
		Region:          viking.Region_BOE,
		EmbeddingConfig: vikingdb.EmbeddingConfig{Embedding: emb},
		SubIndexes:      nil,
		AddBatchSize:    5,
	})

	if err != nil {
		logs.CtxError(ctx, "NewIndexer failed, err=%v", err)
		return
	}

	doc := &schema.Document{Content: "A ReAct prompt consists of few-shot task-solving trajectories, with human-written text reasoning traces and actions, as well as environment observations in response to actions"}

	resp, err := cli.Store(ctx, []*schema.Document{doc})
	if err != nil {
		logs.CtxError(ctx, "vikingDB store failed, err=%v", err)
		return
	}

	logs.CtxInfo(ctx, "vikingDB store success, ids=%v", resp)
}

type mockEmbedding struct{}

func (m mockEmbedding) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	return [][]float64{{1.1}}, nil
}
