package vikingdb

const typ = "VikingDB"

func f64To32(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, f := range f64 {
		f32[i] = float32(f)
	}

	return f32
}

func f32To64(f32 []float32) []float64 {
	f64 := make([]float64, len(f32))
	for i, f := range f32 {
		f64[i] = float64(f)
	}

	return f64
}

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
