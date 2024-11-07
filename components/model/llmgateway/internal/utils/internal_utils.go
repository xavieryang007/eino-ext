package utils

import (
	"code.byted.org/gopkg/lang/conv"
	"fmt"
)

func EmptyIfNil(v *string) string {
	if v != nil {
		return *v
	}
	return ""
}

func F64Ptr(v *float32) *float64 {
	if v != nil {
		s := fmt.Sprintf("%f", *v)
		if f, e := conv.Float64(s); e == nil {
			return &f
		}

		val := float64(*v)
		return &val
	}

	return nil
}

func I64Ptr(v *int) *int64 {
	if v != nil {
		val := int64(*v)
		return &val
	}

	return nil
}

func IntPtr(v *int64) *int {
	if v != nil {
		val := int(*v)
		return &val
	}

	return nil
}

func toStringPtr(v string) *string {
	if v == "" {
		return nil
	}

	return &v
}
