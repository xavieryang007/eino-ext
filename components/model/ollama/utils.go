package ollama

func ptrOf[T any](v T) *T {
	return &v
}
