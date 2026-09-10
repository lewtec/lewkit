package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsagePositionals(t *testing.T) {
	type args struct {
		path StringArg `help:"file to read"`
		rest []StringArg
	}
	text, err := Usage[args]("tool")
	require.NoError(t, err)
	assert.Contains(t, text, "tool [flags] <arg> [arg...]")
	assert.Contains(t, text, "Arguments:")
}
