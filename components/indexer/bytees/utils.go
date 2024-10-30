package bytees

import "code.byted.org/flow/eino/schema"

func toESDoc(doc *schema.Document) map[string]any {
	mp := make(map[string]any)
	if kvs, ok := getExtraDataFields(doc); ok {
		for k, v := range kvs {
			mp[k] = v
		}
	}

	mp[DocFieldNameContent] = doc.Content

	return mp
}

func iter[T, D any](src []T, fn func(int, T) D) []D {
	resp := make([]D, len(src))
	for i := range src {
		resp[i] = fn(i, src[i])
	}

	return resp
}

func iterWithErr[T, D any](src []T, fn func(int, T) (D, error)) ([]D, error) {
	resp := make([]D, len(src))
	for i := range src {
		v, err := fn(i, src[i])
		if err != nil {
			return nil, err
		}

		resp[i] = v
	}

	return resp, nil
}
