// Package graphviz is one Graphviz source file.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: graph, node, edge, subgraph, and attribute. [Take] parses
// the source. [Make] prints it. [Take], then [Make], then [Take] produce
// the same file.
//
// A file is one or more graphs. A graph holds statements in order:
// nodes, edges, default attributes, assignments, and subgraphs. A node
// id keeps a port and a compass. An attribute value keeps HTML when the
// source used an HTML string. x/graph.Graph stays a projection.
//
// The contract is in DESIGN.md beside the parent directory.
package graphviz
