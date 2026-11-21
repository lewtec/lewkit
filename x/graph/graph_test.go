package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func sampleGraph() Graph {
	return Graph{
		Title: "demo",
		Nodes: []Node{
			{ID: "s0", Label: "0", Start: true},
			{ID: "s1", Label: "1"},
		},
		Edges: []Edge{
			{From: "s0", To: "s1", Label: `say "hi"`},
			{From: "s0", To: "s0", Label: "ε"},
		},
	}
}

func TestDOT(t *testing.T) {
	got := sampleGraph().DOT()
	assert.Contains(t, got, "digraph {")
	assert.Contains(t, got, "rankdir=LR;")
	assert.Contains(t, got, `label="demo";`)
	assert.Contains(t, got, `"s0" [label="0", shape=doublecircle];`)
	assert.Contains(t, got, `"s1" [label="1", shape=circle];`)
	assert.Contains(t, got, `"s0" -> "s1" [label="say \"hi\""];`)
	assert.Contains(t, got, `"s0" -> "s0" [label="ε"];`)
}

func TestMermaid(t *testing.T) {
	got := sampleGraph().Mermaid()
	assert.Contains(t, got, "flowchart LR")
	assert.Contains(t, got, "%% demo")
	assert.Contains(t, got, `s0(("0"))`)
	assert.Contains(t, got, `s1("1")`)
	assert.Contains(t, got, `s0 -->|"say #quot;hi#quot;"| s1`)
	assert.Contains(t, got, `s0 -->|"ε"| s0`)
}

func TestEmptyGraph(t *testing.T) {
	assert.Equal(t, "digraph {\n\trankdir=LR;\n}\n", Graph{}.DOT())
	assert.Equal(t, "flowchart LR\n", Graph{}.Mermaid())
}
