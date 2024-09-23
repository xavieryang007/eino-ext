package vikingdb

import (
	"errors"
	"strconv"
)

const typ = "VikingDB"

func parseUint64(idStr string) (uint64, error) {
	if len(idStr) == 0 {
		return 0, errors.New("invalid empty id str")
	}

	ui64, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, err
	}

	return ui64, nil
}

func f64To32(f64 []float64) []float32 {
	f32 := make([]float32, len(f64))
	for i, f := range f64 {
		f32[i] = float32(f)
	}

	return f32
}
