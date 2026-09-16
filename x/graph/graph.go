// Package graph is a directed graph that prints as DOT or Mermaid.
package graph

// Graph is a directed graph. Node IDs must be unique and, for Mermaid,
// start with a letter.
type Graph struct {
	Title string
	Nodes []Node
	Edges []Edge
}

// Node is a vertex. Start marks the entry of an automaton.
type Node struct {
	ID    string
	Label string
	Start bool
}

// Edge is a directed, labeled arc.
type Edge struct {
	From  string
	To    string
	Label string
}
