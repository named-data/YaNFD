package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/exp/constraints"
)

var testT *testing.T

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}
	return b
}

func SetTestingT(t *testing.T) {
	testT = t
}

func WithoutErr[T any](v T, err error) T {
	require.NoError(testT, err)
	return v
}

func WithErr[T any](v T, err error) error {
	require.Error(testT, err)
	return err
}
