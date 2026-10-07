package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/tool"
	"github.com/stretchr/testify/require"
)

func TestCondaBackendRegistered(t *testing.T) {
	backend, err := tool.Get("conda")
	require.NoError(t, err)
	require.Equal(t, "conda", backend.Name())
}
