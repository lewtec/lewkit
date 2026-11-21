package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphCommands(t *testing.T) {
	drawn, err := Graph[appArgs]()
	require.NoError(t, err)
	labels := make([]string, len(drawn.Edges))
	for i, edge := range drawn.Edges {
		labels[i] = edge.Label
	}
	assert.Contains(t, labels, "add")
	assert.Contains(t, labels, "rm")
	assert.Contains(t, labels, "--verbose -v")
	assert.Contains(t, labels, "--")
	var start bool
	for _, node := range drawn.Nodes {
		if node.Start {
			start = true
		}
	}
	assert.True(t, start)
	assert.Contains(t, drawn.DOT(), "add")
	assert.Contains(t, drawn.Mermaid(), "add")
}

func TestGraphInvalid(t *testing.T) {
	_, err := Graph[mixedArgs]()
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

func TestGraphEnumChoices(t *testing.T) {
	type args struct {
		color EnumArg[color]
	}
	drawn, err := Graph[args]()
	require.NoError(t, err)
	labels := make([]string, len(drawn.Edges))
	for i, edge := range drawn.Edges {
		labels[i] = edge.Label
	}
	assert.Contains(t, labels, "red|green|blue")
}
