package utils

func ConstPtr[T any](value T) *T {
	return &value
}
