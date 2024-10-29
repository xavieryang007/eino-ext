package bytedgpt

const typ = "OpenAI"

func getType() string {
	return typ
}

func dereferenceOrZero[T any](v *T) T {
	if v == nil {
		var t T
		return t
	}

	return *v
}
