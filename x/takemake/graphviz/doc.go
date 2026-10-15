// Package graphviz is one Graphviz source file.
//
// [New] returns an empty [File]. Methods return the receiver and store
// the child: graph, node, edge, and attribute. [Take] parses the source.
// [Make] prints it. A projection to x/graph.Graph is separate. The
// contract is in DESIGN.md beside the parent directory. This scaffold
// returns [github.com/lewtec/lewkit/x/errs.ErrScaffold].
package graphviz
