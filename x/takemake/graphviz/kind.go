package graphviz

// StmtKind is one statement in a graph.
// The zero value is unset.
//
//lewkit:enum invalid=unset
type StmtKind uint8

const (
	StmtNode     StmtKind = iota + 1 // node
	StmtEdge                         // edge
	StmtAttr                         // attr
	StmtAssign                       // assign
	StmtSubgraph                     // subgraph
)

// AttrKind is the target of a default attribute statement.
// The zero value is unset.
//
//lewkit:enum invalid=unset
type AttrKind uint8

const (
	AttrGraph AttrKind = iota + 1 // graph
	AttrNode                      // node
	AttrEdge                      // edge
)
