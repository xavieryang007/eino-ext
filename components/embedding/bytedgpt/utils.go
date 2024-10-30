package bytedgpt

func dereferenceOrZero[T any](v *T) T {
	if v == nil {
		var t T
		return t
	}

	return *v
}

func dereferenceOrDefault[T any](v *T, d T) T {
	if v == nil {
		return d
	}

	return *v
}
