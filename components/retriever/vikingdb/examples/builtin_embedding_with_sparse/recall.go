package main

import (
	"context"
	"os"

	"code.byted.org/gopkg/ctxvalues"
	"code.byted.org/gopkg/logid"
	"code.byted.org/gopkg/logs/v2"
	viking "code.byted.org/lagrange/viking_go_client"

	"code.byted.org/flow/eino-ext/components/retriever/vikingdb"
)

func main() {
	vikingDBName := os.Getenv("VIKING_DB_NAME")
	vikingDBToken := os.Getenv("VIKING_DB_TOKEN")

	ctx := ctxvalues.SetLogID(context.Background(), logid.GenLogID())

	baseTopK := 5

	// 混合向量检索: https://bytedance.larkoffice.com/wiki/F3iJwhUa9i3WYFkdOOUcEddFnAh
	// 1. 向量库打开 has_sparse 开关
	// 2. 索引 op_name 选 hnsw
	// 3. 只有部分内置 embedding 输出 sparse vector: https://bytedance.larkoffice.com/wiki/UhCPwrAogi4p2Ukhb9dc74AInSh
	retriever, err := vikingdb.NewRetriever(ctx, &vikingdb.RetrieverConfig{
		Name:   vikingDBName,
		Token:  vikingDBToken,
		Region: viking.Region_BOE,
		EmbeddingConfig: vikingdb.EmbeddingConfig{
			UseBuiltin:       true,
			UseSparse:        true,
			SparseLogitAlpha: 0.5,
		},
		Index:    "3", // index version, replace if needed
		SubIndex: "",
		TopK:     &baseTopK,
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
