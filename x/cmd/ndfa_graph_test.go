package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func graphLabels(t *testing.T, automaton *NDFA) []string {
	t.Helper()
	edges := automaton.Graph().Edges
	labels := make([]string, len(edges))
	for i, edge := range edges {
		labels[i] = edge.Label
	}
	return labels
}

func TestCompileNDFACommands(t *testing.T) {
	automaton, err := CompileNDFA[appArgs]()
	require.NoError(t, err)
	require.NotNil(t, automaton)
	labels := graphLabels(t, automaton)
	assert.Contains(t, labels, "add")
	assert.Contains(t, labels, "rm")
	assert.Contains(t, labels, "--verbose -v")
	assert.Contains(t, labels, "--")
	drawn := automaton.Graph()
	assert.True(t, drawn.Nodes[automaton.Start].Start)
	assert.Contains(t, drawn.DOT(), "add")
	assert.Contains(t, drawn.Mermaid(), "add")
}

func TestCompileNDFAInvalid(t *testing.T) {
	_, err := CompileNDFA[mixedArgs]()
	assert.ErrorIs(t, err, ErrInvalidSpec)
}

func TestNDFAGraphEnumChoices(t *testing.T) {
	type args struct {
		color EnumArg[color]
	}
	automaton, err := CompileNDFA[args]()
	require.NoError(t, err)
	assert.Contains(t, graphLabels(t, automaton), "red|green|blue")
}

func TestNDFAGraphNil(t *testing.T) {
	var automaton *NDFA
	assert.Empty(t, automaton.Graph().Nodes)
}
