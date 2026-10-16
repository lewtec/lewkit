package graphviz

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/errs"
	"github.com/stretchr/testify/require"
)

func TestComposeRoundTrip(t *testing.T) {
	t.Parallel()
	file := sample()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	require.Equal(t, sampleDOT, buf.String())

	taken, err := Take(strings.NewReader(buf.String()))
	require.NoError(t, err)
	require.Equal(t, file, taken)

	var again bytes.Buffer
	require.NoError(t, taken.Make(&again))
	require.Equal(t, sampleDOT, again.String())
}

func TestStoresChildren(t *testing.T) {
	t.Parallel()
	file := New()
	graph := file.Digraph("G")
	node := graph.Node("start").Attr("shape", "doublecircle").Port("f0").Compass("e")
	edge := graph.Edge("start", "next").Attr("label", "go").To("done")
	sub := graph.Subgraph("cluster")
	group := graph.Group()
	attr := graph.NodeAttr("shape", "box").Attr("color", "black")

	require.Same(t, graph, file.Graphs[0])
	require.True(t, graph.Directed)
	require.Same(t, node, graph.Stmts[0].Node)
	require.Equal(t, "f0", node.IDs[0].Port.Text)
	require.Equal(t, "e", node.IDs[0].Compass.Text)
	require.Same(t, edge, graph.Stmts[1].Edge)
	require.Len(t, edge.Ends, 3)
	require.Same(t, sub, graph.Stmts[2].Subgraph)
	require.False(t, sub.Bare)
	require.Same(t, group, graph.Stmts[3].Subgraph)
	require.True(t, group.Bare)
	require.Same(t, attr, graph.Stmts[4].Attr)
	require.Equal(t, AttrNode, attr.Kind)
	require.Equal(t, "black", attr.Attrs[1].Value.Text)
}

func TestProjectionDOT(t *testing.T) {
	t.Parallel()
	// x/graph.Graph.DOT prints this digraph. The projection stays there.
	const src = "digraph {\n" +
		"\trankdir=LR;\n" +
		"\tlabel=\"demo\";\n" +
		"\t\"s0\" [label=\"0\", shape=doublecircle];\n" +
		"\t\"s1\" [label=\"1\", shape=circle];\n" +
		"\t\"s0\" -> \"s1\" [label=\"say \\\"hi\\\"\"];\n" +
		"\t\"s0\" -> \"s0\" [label=\"ε\"];\n" +
		"}\n"
	file, err := Take(strings.NewReader(src))
	require.NoError(t, err)
	graph := file.Graphs[0]
	require.True(t, graph.Directed)
	require.Empty(t, graph.Name.Text)
	require.Equal(t, "LR", assignValue(t, graph, "rankdir"))
	require.Equal(t, "demo", assignValue(t, graph, "label"))
	start := nodeNamed(t, graph, "s0")
	require.Equal(t, "doublecircle", attrValue(start.Attrs, "shape"))
	edge := edgeNamed(t, graph, "s0", "s1")
	require.Equal(t, `say "hi"`, attrValue(edge.Attrs, "label"))
	loop := edgeNamed(t, graph, "s0", "s0")
	require.Equal(t, "ε", attrValue(loop.Attrs, "label"))
	roundTrip(t, file)
}

func TestStringsPortsAndOperators(t *testing.T) {
	t.Parallel()
	file, err := Take(strings.NewReader("digraph {\n" +
		"\ta [label=\"hello\\nworld\"];\n" +
		"\tb [label=\"hello\nworld\"];\n" +
		"\tc [label=\"foo\" + \"bar\"];\n" +
		"\ta:port:sw -> b:e;\n" +
		"\tx, y -> z;\n" +
		"\tnode macro = [shape=box];\n" +
		"}\n"))
	require.NoError(t, err)
	graph := file.Graphs[0]
	require.Equal(t, `hello\nworld`, attrValue(nodeNamed(t, graph, "a").Attrs, "label"))
	got := attrValue(nodeNamed(t, graph, "b").Attrs, "label")
	require.Equal(t, "hello\nworld", got)
	require.Equal(t, "foobar", attrValue(nodeNamed(t, graph, "c").Attrs, "label"))
	require.False(t, findAttr(nodeNamed(t, graph, "c").Attrs, "label").Value.HTML)
	edge := edgeNamed(t, graph, "a", "b")
	require.Equal(t, "port", edge.Ends[0].IDs[0].Port.Text)
	require.Equal(t, "sw", edge.Ends[0].IDs[0].Compass.Text)
	require.True(t, edge.Ends[1].IDs[0].HasPort)
	require.Equal(t, "e", edge.Ends[1].IDs[0].Port.Text)
	require.False(t, edge.Ends[1].IDs[0].HasCompass)
	list := edgeNamed(t, graph, "x", "z")
	require.Equal(t, "y", list.Ends[0].IDs[1].Name.Text)
	require.Equal(t, AttrNode, graph.Stmts[len(graph.Stmts)-1].Attr.Kind)
	require.Equal(t, "box", graph.Stmts[len(graph.Stmts)-1].Attr.Attrs[0].Value.Text)
	roundTrip(t, file)

	mixed, err := Take(strings.NewReader("digraph A { a -> b } graph B { c -- d }\n"))
	require.NoError(t, err)
	require.True(t, mixed.Graphs[0].Directed)
	require.False(t, mixed.Graphs[1].Directed)
	require.Equal(t, "B", mixed.Graphs[1].Name.Text)
	roundTrip(t, mixed)

	strict, err := Take(strings.NewReader("strict graph G { a -- b }\n"))
	require.NoError(t, err)
	require.True(t, strict.Graphs[0].Strict)
	roundTrip(t, strict)

	empty, err := Take(strings.NewReader(""))
	require.NoError(t, err)
	require.Empty(t, empty.Graphs)

	bom, err := Take(strings.NewReader("\uFEFF// note\n# shell\ndigraph { a -> b /* mid */ }\n"))
	require.NoError(t, err)
	require.Equal(t, "a", bom.Graphs[0].Stmts[0].Edge.Ends[0].IDs[0].Name.Text)
	roundTrip(t, bom)
}

func TestSyntax(t *testing.T) {
	t.Parallel()
	_, err := Take(nil)
	require.ErrorIs(t, err, ErrRead)
	require.ErrorIs(t, err, errs.ErrInvalid)

	require.ErrorIs(t, (*File)(nil).Make(io.Discard), ErrFile)
	require.ErrorIs(t, New().Make(nil), ErrWrite)
	require.ErrorIs(t, New().Make(nil), errs.ErrMissing)

	bad := New()
	bad.Digraph("G").Node("a").Compass("n")
	require.ErrorIs(t, bad.Make(io.Discard), ErrForm)
	require.ErrorIs(t, bad.Make(io.Discard), errs.ErrUnsupported)

	for _, src := range []string{
		"digraph { a -- b }",
		"graph { a -> b }",
		"digraph { a [label=\"hi] }",
		"digraph { /* }",
		"not a graph",
		"digraph { a: }",
	} {
		_, err := Take(strings.NewReader(src))
		require.ErrorIs(t, err, ErrSyntax, src)
		require.ErrorIs(t, err, errs.ErrSyntax, src)
	}
}

// Official examples are copied from
// https://gitlab.com/graphviz/graphviz/-/tree/main/graphs (EPL-2.0).
func TestOfficialRoundTrip(t *testing.T) {
	t.Parallel()
	entries, err := os.ReadDir("testdata")
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(entries), 16)
	for _, entry := range entries {
		t.Run(entry.Name(), func(t *testing.T) {
			t.Parallel()
			file := takeFile(t, entry.Name())
			require.NotEmpty(t, file.Graphs)
			require.NotEmpty(t, file.Graphs[0].Stmts)
		})
	}
}

func TestFSM(t *testing.T) {
	t.Parallel()
	graph := takeFile(t, "fsm.gv").Graphs[0]
	require.Equal(t, "finite_state_machine", graph.Name.Text)
	require.True(t, graph.Directed)
	require.Equal(t, "doublecircle", attrValue(graph.Stmts[0].Attr.Attrs, "shape"))
	require.Equal(t, "LR_0", graph.Stmts[1].Node.IDs[0].Name.Text)
	require.Equal(t, "LR_8", graph.Stmts[4].Node.IDs[0].Name.Text)
	require.Equal(t, "LR", assignValue(t, graph, "rankdir"))
	edge := edgeNamed(t, graph, "LR_0", "LR_2")
	require.Equal(t, "SS(B)", attrValue(edge.Attrs, "label"))
	require.Equal(t, 14, countKind(graph, StmtEdge))
}

func TestClusters(t *testing.T) {
	t.Parallel()
	graph := takeFile(t, "clust.gv").Graphs[0]
	require.Equal(t, "cluster_0", graph.Stmts[0].Subgraph.Name.Text)
	require.Equal(t, "hello world", assignValue(t, graph.Stmts[0].Subgraph, "label"))
	require.Equal(t, "cluster_1", graph.Stmts[1].Subgraph.Name.Text)
	require.Equal(t, 9, countKind(graph, StmtEdge))

	chain := takeFile(t, "clust1.gv").Graphs[0]
	first := chain.Stmts[0].Subgraph.Stmts[0].Edge
	require.Len(t, first.Ends, 4)
	require.Equal(t, "a0", first.Ends[0].IDs[0].Name.Text)
	require.Equal(t, "a3", first.Ends[3].IDs[0].Name.Text)
}

func TestPortsRecordsAndHTML(t *testing.T) {
	t.Parallel()
	structs := takeFile(t, "structs.gv").Graphs[0]
	edge := edgeNamed(t, structs, "struct1", "struct2")
	require.Equal(t, "f1", edge.Ends[0].IDs[0].Port.Text)
	require.Equal(t, "f0", edge.Ends[1].IDs[0].Port.Text)
	label := attrValue(nodeNamed(t, structs, "struct3").Attrs, "label")
	require.Contains(t, label, `\n`)
	require.False(t, findAttr(nodeNamed(t, structs, "struct3").Attrs, "label").Value.HTML)

	records := takeFile(t, "records.gv").Graphs[0]
	ports := edgeNamed(t, records, "a", "b")
	require.Equal(t, "bala", ports.Ends[0].IDs[0].Port.Text)
	require.Equal(t, "left", ports.Ends[1].IDs[0].Port.Text)
	quoted := edgeNamed(t, records, "a", "y")
	require.Equal(t, "p1", quoted.Ends[1].IDs[0].Port.Text)
	require.Contains(t, attrValue(nodeNamed(t, records, "a").Attrs, "label"), `\l`)

	table := takeFile(t, "table.gv").Graphs[0]
	html := findAttr(nodeNamed(t, table, "struct1").Attrs, "label").Value
	require.True(t, html.HTML)
	require.Contains(t, html.Text, "<TABLE")
	require.Contains(t, html.Text, "</TABLE>")
}

func TestSubgraphsNumbersAndUndirected(t *testing.T) {
	t.Parallel()
	dfa := takeFile(t, "dfa.gv").Graphs[0]
	sub := subgraphNamed(t, dfa, "l")
	require.Equal(t, "same", assignValue(t, sub, "rank"))
	require.Equal(t, "n3", sub.Stmts[1].Node.IDs[0].Name.Text)

	sw := takeFile(t, "switch.gv").Graphs[0]
	foundSub := false
	walk(sw, func(stmt *Stmt) {
		if stmt.Kind == StmtEdge {
			for _, end := range stmt.Edge.Ends {
				if end.Sub != nil {
					foundSub = true
				}
			}
		}
	})
	require.True(t, foundSub)

	abstract := takeFile(t, "abstract.gv").Graphs[0]
	number := edgeNamed(t, abstract, "S24", "27")
	require.Equal(t, "27", number.Ends[1].IDs[0].Name.Text)

	unix := takeFile(t, "unix.gv").Graphs[0]
	plus := edgeNamed(t, unix, "7th Edition", "UniPlus+")
	require.Equal(t, "UniPlus+", plus.Ends[1].IDs[0].Name.Text)

	world := takeFile(t, "world.gv").Graphs[0]
	into := edgeNamed(t, world, "2", "")
	require.NotNil(t, into.Ends[1].Sub)
	require.True(t, into.Ends[1].Sub.Bare)
	require.Equal(t, "3", into.Ends[1].Sub.Stmts[0].Node.IDs[0].Name.Text)

	er := takeFile(t, "ER.gv").Graphs[0]
	require.False(t, er.Directed)
	link := edgeNamed(t, er, "course", "C-I")
	require.Equal(t, "n", attrValue(link.Attrs, "label"))
	require.Equal(t, "1.00", attrValue(link.Attrs, "len"))

	petersen := takeFile(t, "Petersen.gv").Graphs[0]
	require.Equal(t, "Petersen", petersen.Name.Text)
	var wheel *Edge
	walk(petersen, func(stmt *Stmt) {
		if stmt.Kind == StmtEdge && len(stmt.Edge.Ends) == 6 && endName(stmt.Edge.Ends[0]) == "0" {
			wheel = stmt.Edge
		}
	})
	require.NotNil(t, wheel)
	require.Equal(t, "0", wheel.Ends[5].IDs[0].Name.Text)
	require.Equal(t, "2.6", attrValue(wheel.Attrs, "len"))
	require.Equal(t, `\N`, attrValue(nodeDefaults(t, petersen).Attrs, "label"))
}

func TestMoreOfficialGraphs(t *testing.T) {
	t.Parallel()
	arrows := takeFile(t, "arrows.gv").Graphs[0]
	box := edgeNamed(t, arrows, "_box", "box")
	require.Equal(t, "box", attrValue(box.Attrs, "arrowhead"))
	require.Greater(t, countKind(arrows, StmtEdge), 40)

	grammar := takeFile(t, "grammar.gv").Graphs[0]
	var plus string
	walk(grammar, func(stmt *Stmt) {
		if stmt.Kind != StmtNode {
			return
		}
		for _, id := range stmt.Node.IDs {
			if id.Name.Text == "n4" {
				if label := attrValue(stmt.Node.Attrs, "label"); label != "" {
					plus = label
				}
			}
		}
	})
	require.Equal(t, "+", plus)

	nan := takeFile(t, "NaN.gv").Graphs[0]
	edgeNamed(t, nan, "AbstractMemory", "Memory")
	require.Greater(t, countKind(nan, StmtEdge), 40)
}

func sample() *File {
	file := New()
	graph := file.Digraph("G")
	graph.Assign("rankdir", "LR")
	graph.NodeAttr("shape", "circle")
	graph.Node("start").Attr("shape", "doublecircle").Attr("label", "begin")
	graph.Edge("start", "a").Attr("label", `say "hi"`)
	graph.Edge("struct1", "struct2").FromPort("f1").ToPort("f0")
	sub := graph.Subgraph("cluster_0")
	sub.Assign("label", "process")
	sub.Edge("a", "b").Attr("color", "blue")
	graph.Node("h").HTML("label", "<B>bold</B>")
	return file
}

const sampleDOT = `digraph G {
	rankdir=LR;
	node [shape=circle];
	start [shape=doublecircle, label=begin];
	start -> a [label="say \"hi\""];
	struct1:f1 -> struct2:f0;
	subgraph cluster_0 {
		label=process;
		a -> b [color=blue];
	};
	h [label=<<B>bold</B>>];
}
`

func takeFile(t *testing.T, name string) *File {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	file, err := Take(bytes.NewReader(raw))
	require.NoError(t, err, name)
	roundTrip(t, file)
	return file
}

func roundTrip(t *testing.T, file *File) {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, file.Make(&buf))
	// Buffer.String is the unread portion, so keep the text before Take reads it.
	text := buf.String()
	taken, err := Take(strings.NewReader(text))
	require.NoError(t, err)
	require.Equal(t, file, taken)
	var again bytes.Buffer
	require.NoError(t, taken.Make(&again))
	require.Equal(t, text, again.String())
}

func walk(g *Graph, fn func(*Stmt)) {
	for _, stmt := range g.Stmts {
		fn(stmt)
		switch stmt.Kind {
		case StmtSubgraph:
			walk(stmt.Subgraph, fn)
		case StmtEdge:
			for _, end := range stmt.Edge.Ends {
				if end.Sub != nil {
					walk(end.Sub, fn)
				}
			}
		}
	}
}

func countKind(g *Graph, kind StmtKind) int {
	n := 0
	walk(g, func(stmt *Stmt) {
		if stmt.Kind == kind {
			n++
		}
	})
	return n
}

func assignValue(t *testing.T, g *Graph, name string) string {
	t.Helper()
	var found string
	ok := false
	walk(g, func(stmt *Stmt) {
		if stmt.Kind == StmtAssign && stmt.Assign.Name.Text == name {
			found = stmt.Assign.Value.Text
			ok = true
		}
	})
	require.True(t, ok, name)
	return found
}

func nodeNamed(t *testing.T, g *Graph, name string) *Node {
	t.Helper()
	var found *Node
	walk(g, func(stmt *Stmt) {
		if stmt.Kind != StmtNode {
			return
		}
		for _, id := range stmt.Node.IDs {
			if id.Name.Text == name {
				found = stmt.Node
			}
		}
	})
	require.NotNil(t, found, name)
	return found
}

func edgeNamed(t *testing.T, g *Graph, from, to string) *Edge {
	t.Helper()
	var found *Edge
	walk(g, func(stmt *Stmt) {
		if stmt.Kind != StmtEdge || len(stmt.Edge.Ends) < 2 {
			return
		}
		if endName(stmt.Edge.Ends[0]) == from && endName(stmt.Edge.Ends[len(stmt.Edge.Ends)-1]) == to {
			found = stmt.Edge
		}
	})
	require.NotNil(t, found, from+" -> "+to)
	return found
}

func endName(end *End) string {
	if end == nil || len(end.IDs) == 0 {
		return ""
	}
	return end.IDs[0].Name.Text
}

func subgraphNamed(t *testing.T, g *Graph, name string) *Graph {
	t.Helper()
	var found *Graph
	walk(g, func(stmt *Stmt) {
		if stmt.Kind == StmtSubgraph && stmt.Subgraph.Name.Text == name {
			found = stmt.Subgraph
		}
	})
	require.NotNil(t, found, name)
	return found
}

func nodeDefaults(t *testing.T, g *Graph) *AttrStmt {
	t.Helper()
	for _, stmt := range g.Stmts {
		if stmt.Kind == StmtAttr && stmt.Attr.Kind == AttrNode {
			return stmt.Attr
		}
	}
	require.Fail(t, "node defaults")
	return nil
}

func attrValue(attrs []*Attr, name string) string {
	attr := findAttr(attrs, name)
	if attr == nil {
		return ""
	}
	return attr.Value.Text
}

func findAttr(attrs []*Attr, name string) *Attr {
	for _, attr := range attrs {
		if attr.Name.Text == name {
			return attr
		}
	}
	return nil
}
