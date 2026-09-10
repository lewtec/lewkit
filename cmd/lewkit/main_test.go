package main

import (
	"testing"

	"github.com/lewtec/lewkit/x/cmd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootUsage(t *testing.T) {
	text, err := cmd.Usage[cmd.App[root]]("lewkit")
	require.NoError(t, err)
	assert.Contains(t, text, "Well planned primitives to be used in other projects.")
	assert.Contains(t, text, "log verbosity (default: 0)")
}
