package bytees

const typ = "ByteES"

type Scheme string

const (
	HTTP  Scheme = "http"
	HTTPS Scheme = "https"
)

const (
	defaultBatchSize = 5
)

const (
	opTypeIndex = "index"
)

const (
	docExtraKeyEsFields = "_es_fields" // *schema.Document.MetaData key of es fields except content
)

const (
	// DocFieldNameContent content of *schema.Document, which has two usage.
	DocFieldNameContent = "eino_doc_content"
)
