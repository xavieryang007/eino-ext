package bytees

import (
	"fmt"

	"code.byted.org/flow/eino/schema"
)

// SetExtraDataFields set data fields for es
func SetExtraDataFields(doc *schema.Document, fields map[string]interface{}) {
	if doc == nil {
		return
	}

	if doc.MetaData == nil {
		doc.MetaData = make(map[string]any)
	}

	doc.MetaData[docExtraKeyEsFields] = fields
}

// getExtraDataFields get data fields from *schema.Document
func getExtraDataFields(doc *schema.Document) (fields map[string]interface{}, ok bool) {
	if doc == nil || doc.MetaData == nil {
		return nil, false
	}

	fields, ok = doc.MetaData[docExtraKeyEsFields].(map[string]interface{})

	return fields, ok
}

type VectorFieldKV struct {
	key       VectorFieldKey
	fieldName FieldName
}

// DefaultVectorFieldKV build default VectorFieldKV by fieldName
// docFieldName should be DocFieldNameContent or key got from getExtraDataFields
func DefaultVectorFieldKV(docFieldName FieldName) VectorFieldKV {
	return VectorFieldKV{
		key:       VectorFieldKey(fmt.Sprintf("vector_%s", docFieldName)),
		fieldName: docFieldName,
	}
}

// VectorFieldKey es field for vectorized raw field
type VectorFieldKey string

func (v VectorFieldKey) Field(fieldName FieldName) VectorFieldKV {
	return VectorFieldKV{
		key:       v,
		fieldName: fieldName,
	}
}

type FieldName string

func (v FieldName) Find(doc *schema.Document) (string, bool) {
	if v == DocFieldNameContent {
		return doc.Content, true
	}

	kvs, ok := getExtraDataFields(doc)
	if !ok {
		return "", false
	}

	s, ok := kvs[string(v)].(string)
	return s, ok
}
