package graph

import (
	"fmt"
	"strings"
)

// DOT is the Graphviz source for g.
func (g Graph) DOT() string {
	var b strings.Builder
	b.WriteString("digraph {\n")
	b.WriteString("\trankdir=LR;\n")
	if g.Title != "" {
		fmt.Fprintf(&b, "\tlabel=%s;\n", quoteDOT(g.Title))
	}
	for _, node := range g.Nodes {
		shape := "circle"
		if node.Start {
			shape = "doublecircle"
		}
		fmt.Fprintf(&b, "\t%s [label=%s, shape=%s];\n", quoteDOT(node.ID), quoteDOT(node.Label), shape)
	}
	for _, edge := range g.Edges {
		fmt.Fprintf(&b, "\t%s -> %s [label=%s];\n", quoteDOT(edge.From), quoteDOT(edge.To), quoteDOT(edge.Label))
	}
	b.WriteString("}\n")
	return b.String()
}

func quoteDOT(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	return `"` + s + `"`
}
