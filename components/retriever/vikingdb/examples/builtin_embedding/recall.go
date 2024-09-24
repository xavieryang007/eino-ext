package main

import (
	"context"
	"os"

	"code.byted.org/flow/eino-ext/components/retriever/vikingdb"
	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"
	viking "code.byted.org/lagrange/viking_go_client"
)

func main() {
	vikingDBName := os.Getenv("VIKING_DB_NAME")
	vikingDBToken := os.Getenv("VIKING_DB_TOKEN")

	ctx := ctxvalues.SetLogID(context.Background(), logid.GenLogID())

	retriever, err := vikingdb.NewRetriever(ctx, &vikingdb.RetrieverConfig{
		Name:   vikingDBName,
		Token:  vikingDBToken,
		Region: viking.Region_BOE,
		EmbeddingConfig: vikingdb.EmbeddingConfig{
			UseBuiltin: true,
		},
		Index:    "3", // index version, replace if needed
		SubIndex: "",
		TopK:     5,
	})
	if err != nil {
		logs.CtxError(ctx, "NewRetriever failed, err=%v", err)
		return
	}

	resp, err := retriever.Retrieve(ctx, "tourist attraction")
	if err != nil {
		logs.CtxError(ctx, "vikingDB Retrieve failed, err=%v", err)
		return
	}

	logs.CtxInfo(ctx, "vikingDB Retrieve success, docs=%v", resp)
}
