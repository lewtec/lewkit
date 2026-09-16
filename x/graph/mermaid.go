package graph

import (
	"fmt"
	"strings"
)

// Mermaid is a flowchart LR diagram for g.
func (g Graph) Mermaid() string {
	var b strings.Builder
	b.WriteString("flowchart LR\n")
	if g.Title != "" {
		fmt.Fprintf(&b, "\t%%%% %s\n", strings.ReplaceAll(g.Title, "\n", " "))
	}
	for _, node := range g.Nodes {
		label := quoteMermaid(node.Label)
		if node.Start {
			fmt.Fprintf(&b, "\t%s((%s))\n", node.ID, label)
		} else {
			fmt.Fprintf(&b, "\t%s(%s)\n", node.ID, label)
		}
	}
	for _, edge := range g.Edges {
		fmt.Fprintf(&b, "\t%s -->|%s| %s\n", edge.From, quoteMermaid(edge.Label), edge.To)
	}
	return b.String()
}

func quoteMermaid(s string) string {
	s = strings.ReplaceAll(s, `"`, "#quot;")
	return `"` + s + `"`
}
