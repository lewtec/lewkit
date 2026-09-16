package cmd

import (
	"strconv"
	"strings"

	"github.com/lewtec/lewkit/x/graph"
)

// Graph is the NDFA as a directed graph for DOT or Mermaid.
func (a *NDFA) Graph() graph.Graph {
	if a == nil {
		return graph.Graph{}
	}
	nodes := make([]graph.Node, len(a.States))
	var edges []graph.Edge
	for state, outgoing := range a.States {
		id := stateID(state)
		nodes[state] = graph.Node{
			ID:    id,
			Label: strconv.Itoa(state),
			Start: state == a.Start,
		}
		for _, e := range outgoing {
			edges = append(edges, graph.Edge{
				From:  id,
				To:    stateID(e.To),
				Label: e.graphLabel(),
			})
		}
	}
	return graph.Graph{Nodes: nodes, Edges: edges}
}

func stateID(state int) string {
	return "s" + strconv.Itoa(state)
}

func (e Edge) graphLabel() string {
	switch e.Kind {
	case EdgeEpsilon:
		return "ε"
	case EdgeLiteral:
		return e.Literal
	case EdgeFlag:
		var parts []string
		if e.Long != "" {
			parts = append(parts, "--"+e.Long)
		}
		if e.Short != 0 {
			parts = append(parts, "-"+string(e.Short))
		}
		return strings.Join(parts, " ")
	case EdgeValue:
		if len(e.Choices) > 0 {
			return strings.Join(e.Choices, "|")
		}
		if e.AcceptAny {
			return "any"
		}
		return "value"
	default:
		return ""
	}
}
