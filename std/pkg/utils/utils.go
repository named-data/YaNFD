package utils

import "golang.org/x/exp/constraints"

// IdPtr is the pointer version of id: 'a->'a
func IdPtr[T any](value T) *T {
	return &value
}

// ConvIntPtr converts an integer pointer to another type
func ConvIntPtr[A, B constraints.Integer](a *A) *B {
	if a == nil {
		return nil
	} else {
		b := B(*a)
		return &b
	}
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
