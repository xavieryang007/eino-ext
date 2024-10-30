package main

import (
	"context"
	"fmt"
	"os"

	"code.byted.org/flow/eino-ext/components/retriever/volc_vikingdb"
)

func main() {
	ctx := context.Background()
	ak := os.Getenv("VOLC_VIKING_DB_AK")
	sk := os.Getenv("VOLC_VIKING_DB_SK")
	collectionName := "eino_test"
	indexName := "test_index_1"

	/*
	 * 下面示例中提前构建了一个名为 eino_test 的数据集 (collection)，并在此数据集上构建了一个名为 test_index_1 的 hnsw-hybrid 索引 (index)
	 * 数据集字段配置为:
	 * 字段名称			字段类型			向量维度
	 * ID				string
	 * vector			vector			1024
	 * sparse_vector 	sparse_vector
	 * content			string
	 * extra_field_1	string
	 *
	 * component 使用时注意:
	 * 1. ID / vector / sparse_vector / content 的字段名称与类型与上方配置一致
	 * 2. vector 向量维度需要与 ModelName 对应的模型所输出的向量维度一致
	 * 3. 部分模型不输出稀疏向量，此时 UseSparse 需要设置为 false，collection 可以不设置 sparse_vector 字段
	 */

	cfg := &volc_vikingdb.RetrieverConfig{
		// https://api-vikingdb.volces.com （华北）
		// https://api-vikingdb.mlp.cn-shanghai.volces.com（华东）
		// https://api-vikingdb.mlp.ap-mya.byteplus.com（海外-柔佛）
		Host:              "api-vikingdb.volces.com",
		Region:            "cn-beijing",
		AK:                ak,
		SK:                sk,
		Scheme:            "https",
		ConnectionTimeout: 0,
		Collection:        collectionName,
		Index:             indexName,
		EmbeddingConfig: volc_vikingdb.EmbeddingConfig{
			UseBuiltin:  true,
			ModelName:   "bge-m3",
			UseSparse:   true,
			DenseWeight: 0.4,
		},
		Partition:      "", // 对应索引中的【子索引划分字段】, 未设置时至空即可
		TopK:           of(10),
		ScoreThreshold: of(0.1),
		FilterDSL:      nil, // 对应索引中的【标量过滤字段】，未设置时至空即可，表达式详见 https://www.volcengine.com/docs/84313/1254609
	}

	ret, err := volc_vikingdb.NewRetriever(ctx, cfg)
	if err != nil {
		fmt.Printf("NewRetriever failed, %v\n", err)
		return
	}

	query := "tourist attraction"
	docs, err := ret.Retrieve(ctx, query)
	if err != nil {
		fmt.Printf("vikingDB retrieve failed, %v\n", err)
		return
	}

	fmt.Printf("vikingDB retrieve success, query=%v, docs=%v\n", query, docs)
}

func of[T any](v T) *T {
	return &v
}
