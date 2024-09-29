package fornaxknowledge

func dereferenceOrZero[T any](v *T) T {
	if v == nil {
		var t T
		return t
	}

	return *v
}

func ptrOf[T any](v T) *T {
	return &v
}
