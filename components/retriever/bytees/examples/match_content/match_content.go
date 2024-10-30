package main

import (
	"context"

	"code.byted.org/flow/eino-ext/components/retriever/bytees"
	"code.byted.org/gopkg/logs"
	"code.byted.org/lang/gg/gptr"
)

func main() {
	ctx := context.Background()
	cfg := &bytees.RetrieverConfig{
		PSM:            "byte.es.knntest_boe",
		Cluster:        "data",
		Index:          "eino_app_es_knn_test_2",
		TopK:           5,
		ScoreThreshold: gptr.Of(0.1),
		SearchMode:     bytees.SearchModeContentMatch,
	}

	ret, err := bytees.NewRetriever(ctx, cfg)
	if err != nil {
		logs.CtxError(ctx, "NewRetriever failed, %v", err)
		return
	}

	query := "长城"
	docs, err := ret.Retrieve(ctx, query) // {"match":{"eino_doc_content":{"query":"$query"}}}
	if err != nil {
		logs.CtxError(ctx, "bytees Retrieve failed, %v", err)
		return
	}

	logs.CtxInfo(ctx, "bytees Retrieve success, result: %v", docs)

	for i, doc := range docs {
		// extra non-content fields by GetExtraDataFields
		ext, ok := bytees.GetExtraDataFields(doc)
		if !ok {
			logs.CtxWarn(ctx, "docs[%d] extra fields not found", i)
		} else {
			logs.CtxInfo(ctx, "docs[%d] extra fields=%+v", i, ext)
		}
	}
}
