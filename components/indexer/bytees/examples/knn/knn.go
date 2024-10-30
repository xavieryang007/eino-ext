package main

import (
	"context"

	"code.byted.org/flow/eino-ext/components/indexer/bytees"
	"code.byted.org/flow/eino/components/embedding"
	"code.byted.org/flow/eino/schema"
	"code.byted.org/gopkg/logs"
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
  - 3. write data
*/

func main() {
	ctx := context.Background()
	dimensions := 1024 // embedding dimension should match index
	extraFieldName := bytees.FieldName("extra_field")
	cfg := &bytees.IndexerConfig{
		PSM:     "byte.es.knntest_boe",
		Cluster: "data",
		Index:   "eino_app_es_knn_test_2",
		UseKNN:  true,
		VectorFields: []bytees.VectorFieldKV{
			// use DefaultVectorFieldKV to generate default vectorization field name, or
			// customize with bytees.VectorFieldKey("my_custom_key").Field(bytees.DocFieldNameContent)
			bytees.DefaultVectorFieldKV(bytees.DocFieldNameContent), // vectorize doc content -> vector_eino_doc_content
			bytees.DefaultVectorFieldKV(extraFieldName),             // vectorize "extra_field" in doc extra -> vector_extra_field
		},
		Embedding: newMockEmbedding(dimensions),
	}

	indexer, err := bytees.NewIndexer(ctx, cfg)
	if err != nil {
		logs.CtxError(ctx, "NewIndexer failed, err=%v", err)
		return
	}

	doc := &schema.Document{
		ID:       "bytees_example_mock_primary_key_0",
		Content:  "bytees_example_mock_content_value",
		MetaData: nil,
	}

	// set extra fields with specified function SetExtraDataFields
	bytees.SetExtraDataFields(doc, map[string]interface{}{
		string(extraFieldName): "bytees_example_mock_extra_field_value",
	})

	ids, err := indexer.Store(ctx, []*schema.Document{doc})
	if err != nil {
		logs.CtxError(ctx, "bytees store failed, err=%v", err)
		return
	}

	logs.CtxInfo(ctx, "bytees store success, ids=%v", ids)
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
