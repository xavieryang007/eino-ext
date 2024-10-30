package volc_vikingdb

import "code.byted.org/flow/eino/schema"

// SetExtraDataFields set data fields for vikingdb UpsertData
// see: https://www.volcengine.com/docs/84313/1254578
func SetExtraDataFields(doc *schema.Document, fields map[string]interface{}) {
	if doc == nil {
		return
	}

	if doc.MetaData == nil {
		doc.MetaData = make(map[string]any)
	}

	doc.MetaData[extraKeyVikingDBFields] = fields
}

// SetExtraDataTTL set data ttl for vikingdb UpsertData
// see: https://www.volcengine.com/docs/84313/1254578
func SetExtraDataTTL(doc *schema.Document, ttl int64) {
	if doc == nil {
		return
	}

	if doc.MetaData == nil {
		doc.MetaData = make(map[string]any)
	}

	doc.MetaData[extraKeyVikingDBTTL] = ttl
}

func GetExtraVikingDBFields(doc *schema.Document) (map[string]interface{}, bool) {
	if doc == nil || doc.MetaData == nil {
		return nil, false
	}

	val, ok := doc.MetaData[extraKeyVikingDBFields].(map[string]interface{})
	return val, ok
}

func GetExtraVikingDBTTL(doc *schema.Document) (int64, bool) {
	if doc == nil || doc.MetaData == nil {
		return 0, false
	}

	val, ok := doc.MetaData[extraKeyVikingDBTTL].(int64)
	return val, ok
}
