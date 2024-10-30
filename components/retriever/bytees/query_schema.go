package bytees

import "encoding/json"

// NewKNNQuery generate knn query for SearchModeKNN and SearchModeKNNWithFilters
func NewKNNQuery(kvs ...VectorFieldKV) (string, error) {
	b, err := json.Marshal(kvs)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

// JoinDSLFilters add filters to dsl in option for SearchModeKNNWithFilters
func JoinDSLFilters(src map[string]interface{}, pairs ...DSLPair) map[string]interface{} {
	if src == nil {
		return src
	}

	src[dslFilterField] = pairs
	return src
}
