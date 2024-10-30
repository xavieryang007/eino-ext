package bytees

import (
	"fmt"

	"code.byted.org/flow/eino/schema"
	tes "code.byted.org/toutiao/elastic/v7"
)

// GetDefaultVectorFieldKeyContent get default es key for Document.Content
func GetDefaultVectorFieldKeyContent() VectorFieldKey {
	return defaultVectorFieldKeyContent
}

// GetDefaultVectorFieldKey generate default vector field name from its field name
func GetDefaultVectorFieldKey(fieldName string) VectorFieldKey {
	return VectorFieldKey(fmt.Sprintf("vector_%s", fieldName))
}

// GetExtraDataFields get data fields from *schema.Document
func GetExtraDataFields(doc *schema.Document) (fields map[string]interface{}, ok bool) {
	if doc == nil || doc.MetaData == nil {
		return nil, false
	}

	fields, ok = doc.MetaData[docExtraKeyEsFields].(map[string]interface{})

	return fields, ok
}

// VectorFieldKey es field for vectorized raw field
type VectorFieldKey string

type VectorFieldKV struct {
	Key   VectorFieldKey `json:"key,omitempty"`
	Value string         `json:"value,omitempty"`
}

type DSLPair struct {
	Query         tes.Query
	ScoreFunction tes.ScoreFunction
}
