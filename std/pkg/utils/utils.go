package utils

import "golang.org/x/exp/constraints"

// IdPtr is the pointer version of id: 'a->'a
func IdPtr[T any](value T) *T {
	return &value
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}
