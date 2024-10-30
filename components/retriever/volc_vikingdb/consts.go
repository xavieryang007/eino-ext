package volc_vikingdb

const typ = "VikingDB"

const (
	ExtraKeyVikingDBFields = "_vikingdb_fields" // value: map[string]interface{}
	ExtraKeyVikingDBTTL    = "_vikingdb_ttl"    // value: int64
)

const (
	defaultFieldContent = "content"
)

const (
	vikingEmbeddingUseDense           = "return_dense"
	vikingEmbeddingUseSparse          = "return_sparse"
	vikingEmbeddingRespSentenceDense  = "sentence_dense_embedding"
	vikingEmbeddingRespSentenceSparse = "sentence_sparse_embedding"
)
