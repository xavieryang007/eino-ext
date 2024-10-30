package main

import (
	"context"

	"code.byted.org/flow/eino-ext/components/retriever/bytees"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/components/retriever"
	"code.byted.org/gopkg/logs"
	"code.byted.org/lang/gg/gptr"
	tes "code.byted.org/toutiao/elastic/v7"
)

/*
  - there are a few preliminary steps to use bytees knn feature
  - 1. install / upgrade plugin，see: https://cloud.bytedance.net/docs/ses/docs/6614e634c6a2250303cdeb04/6630ae80a4561c02e780c33f=
  - 2. create index like example below, notice the schema of embedding field like "vector_eino_doc_content" and "vector_extra_field"
```
	{
	    "settings": {
	        "index.knn.space_type": "l2",
	        "index.knn.algo_param.m": "16",
	        "index.knn.algo_param.ef_search": 512,
	        "index.knn.algo_param.ef_construction": 512,
	        "index.knn": true
	    },
	    "mappings": {
	        "properties": {
	            "eino_doc_content": {
	                "fields": {
	                    "keyword": {
	                        "ignore_above": 256,
	                        "type": "keyword"
	                    }
	                },
	                "type": "text"
	            },
	            "extra_field": {
	                "fields": {
	                    "keyword": {
	                        "ignore_above": 256,
	                        "type": "keyword"
	                    }
	                },
	                "type": "text"
	            },
	            "vector_eino_doc_content": {
	                "dimension": 1024,
	                "type": "knn_vector"
	            },
	            "vector_extra_field": {
	                "dimension": 1024,
	                "type": "knn_vector"
	            }
	        }
	    }
	}
```
  - 3. retrieve data, retrieved result be like:
```
		[
		    {
		        "id": "10002",
		        "content": "2. 长城：位于中国，是世界七大奇迹之一，从秦至明代修筑而成，全长超过 2 万公里。",
		        "meta_data": {
		            "_es_fields": {
		                "extra_field": "中国",
		                "vector_eino_doc_content": Array[1024],
		                "vector_extra_field": Array[1024]
		            },
		            "_score": 10
		        }
		    },
		    {
		        "id": "10008",
		        "content": "8. 尼亚加拉大瀑布：位于美国和加拿大交界处，由三个主要瀑布组成，其壮观的景象每年吸引数百万游客。",
		        "meta_data": {
		            "_es_fields": {
		                "extra_field": "美国",
		                "vector_eino_doc_content": Array[1024],
		                "vector_extra_field": Array[1024]
		            },
		            "_score": 8.70513
		        }
		    }
		]
```
*/

func main() {
	ctx := context.Background()
	dimensions := 1024 // embedding dimension should match index
	cfg := &bytees.RetrieverConfig{
		PSM:            "byte.es.knntest_boe",
		Cluster:        "data",
		Index:          "eino_app_es_knn_test_2",
		TopK:           5,
		ScoreThreshold: gptr.Of(0.1),
		SearchMode:     bytees.SearchModeKNN,
		Embedding:      newMockEmbedding(dimensions), // replace with real embedding
	}

	ret, err := bytees.NewRetriever(ctx, cfg)
	if err != nil {
		logs.CtxError(ctx, "NewRetriever failed, %v", err)
		return
	}

	// query should be json message of []VectorFieldKV when SearchMode is SearchModeKNN
	// use NewKNNQuery to generate proper query rapidly
	query, err := bytees.NewKNNQuery(
		bytees.VectorFieldKV{Key: bytees.GetDefaultVectorFieldKeyContent(), Value: "tourist spots"}, // content vector
		bytees.VectorFieldKV{Key: bytees.GetDefaultVectorFieldKey("extra_field"), Value: "asia"},    // extra_field vector
	)
	if err != nil {
		logs.CtxError(ctx, "NewKNNQuery failed, %v", err)
		return
	}

	// use JoinDSLFilters to customize es filter query and score function
	dsl := bytees.JoinDSLFilters(map[string]interface{}{},
		bytees.DSLPair{
			Query:         tes.NewRawStringQuery("{\"match\":{\"content\":\"中国\"}}"),
			ScoreFunction: tes.NewWeightFactorFunction(10),
		},
	)

	docs, err := ret.Retrieve(ctx, query, retriever.WithDSLInfo(dsl))
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

func newMockEmbedding(dimensions int) *mockEmbedding {
	v := make([]float64, dimensions)
	for i := range v {
		v[i] = 1.1
	}

	return &mockEmbedding{v}
}

type mockEmbedding struct {
	mockVector []float64
}

func (m mockEmbedding) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	resp := make([][]float64, len(texts))
	for i := range resp {
		resp[i] = m.mockVector
	}

	return resp, nil
}
