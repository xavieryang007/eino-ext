package volc_vikingdb

const typ = "VikingDB"

const (
	extraKeyVikingDBFields = "_vikingdb_fields" // value: map[string]interface{}
	extraKeyVikingDBTTL    = "_vikingdb_ttl"    // value: int64
)

const (
	defaultFieldID           = "ID"
	defaultFieldVector       = "vector"
	defaultFieldSparseVector = "sparse_vector"
	defaultFieldContent      = "content"
)

const (
	vikingEmbeddingUseDense           = "return_dense"
	vikingEmbeddingUseSparse          = "return_sparse"
	vikingEmbeddingRespSentenceDense  = "sentence_dense_embedding"
	vikingEmbeddingRespSentenceSparse = "sentence_sparse_embedding"
)
