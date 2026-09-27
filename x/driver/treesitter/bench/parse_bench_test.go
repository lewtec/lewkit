package bench

import (
	"testing"

	_ "github.com/lewtec/leaven-tree-sitter/grammar/json"
	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/treesitter"
	_ "github.com/lewtec/lewkit/x/driver/treesitter/prelude"
	_ "github.com/lewtec/wazero-tree-sitter/grammar/json"
	_ "github.com/modernc-tree-sitter/ccgo-tree-sitter/grammar/json"
	"github.com/stretchr/testify/require"
)

func BenchmarkParse(b *testing.B) {
	b.Setenv("LEWKIT_ENABLE_NATIVE_TREESITTER", "1")
	source := jsonSource()
	for _, id := range []string{
		"treesitter_leaven",
		"treesitter_ccgo",
		"treesitter_wazero",
		"treesitter_native",
	} {
		b.Run(id, func(b *testing.B) {
			lang := language(b, id)
			tree, err := lang.Parse(source)
			require.NoError(b, err)
			requireParsed(b, tree)
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for b.Loop() {
				tree, err = lang.Parse(source)
				require.NoError(b, err)
			}
			requireParsed(b, tree)
		})
	}
}

func language(b *testing.B, id string) treesitter.Language {
	b.Helper()
	handles, err := driver.List[treesitter.Driver](b.Context())
	require.NoError(b, err)
	for _, handle := range handles {
		if handle.ID != id {
			continue
		}
		engine, err := handle.Open(b.Context())
		require.NoError(b, err)
		lang, ok := engine.Language("json")
		if !ok {
			b.Skipf("%s has no json grammar", id)
		}
		return lang
	}
	b.Skipf("%s is unavailable", id)
	return nil
}

func requireParsed(b *testing.B, tree *treesitter.Tree) {
	b.Helper()
	require.NotNil(b, tree)
	root := tree.RootNode()
	require.NotNil(b, root)
	require.False(b, root.IsNull())
	require.False(b, root.HasError())
}

func jsonSource() []byte {
	return []byte(`{"a":1,"b":[true,false,null],"c":"row"}`)
}
