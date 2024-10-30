package bytees

const typ = "ByteES"

type Scheme string

const (
	HTTP  Scheme = "http"
	HTTPS Scheme = "https"
)

const defaultTopK = 10

// SearchMode es search mode, related to retrieve query's schema
type SearchMode int

const (
	// SearchModeContentMatch match content of query only
	SearchModeContentMatch SearchMode = iota

	// SearchModeKNN search by knn
	// Use NewKNNQuery to generate retriever query
	// Use JoinDSLFilters and WithDSLInfo to set filter functions in retrieve Options
	SearchModeKNN

	// SearchModeRawStringQuery search by raw query
	// retrieve query will be used with NewRawStringQuery(query)
	SearchModeRawStringQuery
)

const (
	docExtraKeyEsFields = "_es_fields" // *schema.Document.MetaData key of es fields except content
	dslFilterField      = "_dsl_filter_functions"
)

const (
	DocFieldNameContent = "eino_doc_content"
)

var defaultVectorFieldKeyContent = GetDefaultVectorFieldKey(DocFieldNameContent)

type knnQuery struct {
	KNN map[string]vectorQuery `json:"knn"`
}

type vectorQuery struct {
	Vector []float64 `json:"vector"`
	K      int       `json:"k"`
}
