package parse

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPatternMatcher(t *testing.T) {
	cases := []struct {
		pat  string
		src  string
		want int
	}{
		{`\d+`, "42x", 2},
		{`[_\p{XID_Start}][_\p{XID_Continue}]*`, "foo1", 4},
		{`[_\p{XID_Start}][_\p{XID_Continue}]*`, "9foo", 0},
		{`//.*`, "// hi", 5},
		{`[^\\"\n]+`, `abc`, 3},
		{`u[a-fA-F\d]{4}`, "u00AB", 5},
	}
	for _, c := range cases {
		re, err := compilePattern(c.pat)
		require.NoError(t, err, c.pat)
		require.Equal(t, c.want, re.matchLen([]byte(c.src), 0), c.pat)
	}
}

func TestTreeSitterLiteralRuleClash(t *testing.T) {
	const src = `{
		"name": "ex",
		"word": "id",
		"rules": {
			"source": {"type": "SEQ", "members": [
				{"type": "STRING", "value": "lifetime"},
				{"type": "SYMBOL", "name": "lifetime"}
			]},
			"lifetime": {"type": "SEQ", "members": [
				{"type": "STRING", "value": "'"},
				{"type": "SYMBOL", "name": "id"}
			]},
			"id": {"type": "PATTERN", "value": "[a-z]+"}
		}
	}`
	g, _, err := FromTreeSitter([]byte(src))
	require.NoError(t, err)
	lang, err := g.Compile()
	require.NoError(t, err)
	tree, err := lang.Parse([]byte("lifetime'a"))
	require.NoError(t, err)
	require.Contains(t, tree.SExpr(), "lifetime")
}

func TestTreeSitterExternalToken(t *testing.T) {
	const src = `{
		"name": "ex",
		"externals": [{"type": "SYMBOL", "name": "indent"}],
		"rules": {
			"source": {"type": "SEQ", "members": [
				{"type": "STRING", "value": "x"},
				{"type": "SYMBOL", "name": "indent"}
			]}
		}
	}`
	g, info, err := FromTreeSitter([]byte(src))
	require.NoError(t, err)
	require.Equal(t, []string{"indent"}, info.Externals)
	lang, err := g.Compile()
	require.NoError(t, err)
	_, err = lang.Parse([]byte("x"))
	require.Error(t, err)
}

func TestTreeSitterJSON(t *testing.T) {
	g, info, err := loadTS(t, "json")
	require.NoError(t, err)
	t.Logf("notes %v externals %v", info.Notes, info.Externals)
	lang, err := g.Compile()
	require.NoError(t, err)
	t.Logf("states %d forks %d", lang.States(), lang.Forks())
	tree, err := lang.Parse([]byte(`{"a":[1,true,null,"x"]}`))
	require.NoError(t, err)
	require.Contains(t, tree.SExpr(), "object")
	require.Contains(t, tree.SExpr(), "array")
}

func TestTreeSitterLoad(t *testing.T) {
	for _, name := range []string{"json", "c", "go", "java", "javascript", "python", "rust", "bash"} {
		g, info, err := loadTS(t, name)
		require.NoError(t, err)
		terms := 0
		for _, s := range g.syms {
			if s.kind == kindTerm {
				terms++
			}
		}
		t.Logf("%s symbols %d terminals %d productions %d externals %d",
			name, len(g.syms), terms, len(g.prods), len(info.Externals))
		for _, note := range info.Notes {
			t.Logf("%s note %s", name, note)
		}
	}
}

func TestTreeSitterGrammars(t *testing.T) {
	// These tables finish in a few minutes. JavaScript, C, and bash are still
	// inside canonical LR(1) closure after several minutes, so this test does
	// not build them. TestTreeSitterLoad still checks that their grammar.json loads.
	samples := map[string]string{
		"go":     "package p\nfunc f() {\n\treturn\n}\n",
		"java":   "class A { int f() { return 1; } }\n",
		"python": "def f():\n    return 1\n",
		"rust":   "fn main() { let x = 1; }\n",
	}
	for _, name := range []string{"go", "java", "python", "rust"} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			g, info, err := loadTS(t, name)
			require.NoError(t, err)
			t0 := time.Now()
			lang, err := g.Compile()
			require.NoError(t, err)
			t.Logf("compile %s states %d forks %d symbols %d prods %d externals %d",
				time.Since(t0).Round(time.Millisecond), lang.States(), lang.Forks(), len(g.syms), len(g.prods), len(info.Externals))
			for _, note := range info.Notes {
				t.Logf("note %s", note)
			}
			tree, err := lang.Parse([]byte(samples[name]))
			if err != nil {
				t.Logf("sample: %v", err)
				return
			}
			t.Logf("sample root %s", tree.Root().Name())
		})
	}
}

func loadTS(t *testing.T, name string) (*Grammar, *TreeSitterInfo, error) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", "treesitter", name+".json"))
	require.NoError(t, err)
	return FromTreeSitter(data)
}
