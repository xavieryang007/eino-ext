package bytees

func iter[T, D any](src []T, fn func(int, T) D) []D {
	resp := make([]D, len(src))
	for i := range src {
		resp[i] = fn(i, src[i])
	}

	return resp
}

func getDSLFilterField(dsl map[string]interface{}) ([]DSLPair, bool) {
	if dsl == nil || dsl[dslFilterField] == nil {
		return nil, false
	}

	pair, found := dsl[dslFilterField].([]DSLPair)
	return pair, found
}
