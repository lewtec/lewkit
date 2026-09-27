package bench

import (
	"errors"
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
			lang, err := treesitter.Open(b.Context(), id, "json")
			if errors.Is(err, driver.ErrNotFound) {
				b.Skip(err)
			}
			require.NoError(b, err)
			tree, err := lang.Parse(source)
			require.NoError(b, err)
			require.NoError(b, tree.Parsed())
			b.ReportAllocs()
			b.SetBytes(int64(len(source)))
			for b.Loop() {
				tree, err = lang.Parse(source)
				require.NoError(b, err)
			}
			require.NoError(b, tree.Parsed())
		})
	}
}

func jsonSource() []byte {
	return []byte(`{"a":1,"b":[true,false,null],"c":"row"}`)
}
