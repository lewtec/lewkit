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

type greetCmd struct{}

func (greetCmd) Description() string {
	return "say hello"
}

func TestUsageDescription(t *testing.T) {
	text, err := Usage[greetCmd]("greet")
	require.NoError(t, err)
	assert.Contains(t, text, "say hello")
}

type listedCmd struct{}

func (listedCmd) Description() string {
	return "copy files\n\nlonger body"
}

type listedApp struct {
	copy *listedCmd
}

func TestUsageCommandDescriptionFallback(t *testing.T) {
	text, err := Usage[listedApp]("tool")
	require.NoError(t, err)
	assert.Contains(t, text, "copy")
	assert.Contains(t, text, "copy files")
	assert.NotContains(t, text, "longer body")
}

type defaultArgs struct {
	name StringArg   `long:"name" help:"who" default:"Lucas"`
	port IntArg[int] `long:"port" default:"8080"`
}

func TestUsageDefault(t *testing.T) {
	text, err := Usage[defaultArgs]("tool")
	require.NoError(t, err)
	assert.Contains(t, text, "--name")
	assert.Contains(t, text, "who (default: Lucas)")
	assert.Contains(t, text, "(default: 8080)")
}
