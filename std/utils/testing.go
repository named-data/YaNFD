package utils

import (
	"testing"

	"github.com/stretchr/testify/require"
)

var testT *testing.T

func SetTestingT(t *testing.T) {
	testT = t
}

func WithoutErr[T any](v T, err error) T {
	require.NoError(testT, err)
	return v
}

func WithErr[T any](_ T, err error) error {
	require.Error(testT, err)
	return err
}
