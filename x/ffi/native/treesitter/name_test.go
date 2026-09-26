package treesitter

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/require"
)

func TestGrammarName(t *testing.T) {
	name, ok := grammarName("libtree-sitter-json.so")
	require.True(t, ok)
	require.Equal(t, "json", name)

	name, ok = grammarName("libtree-sitter-c-sharp.so.0.24")
	require.True(t, ok)
	require.Equal(t, "c-sharp", name)

	name, ok = grammarName("/usr/lib/libtree-sitter-go.dylib")
	require.True(t, ok)
	require.Equal(t, "go", name)

	_, ok = grammarName("libtree-sitter.so.0")
	require.False(t, ok)
	_, ok = grammarName("libtree-sitter-.so")
	require.False(t, ok)
}

func TestSymbolName(t *testing.T) {
	require.Equal(t, "tree_sitter_json", symbolName("json"))
	require.Equal(t, "tree_sitter_c_sharp", symbolName("c-sharp"))
}

func TestRawNodeSize(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) != 8 {
		t.Skip("TSNode layout is asserted on 64-bit")
	}
	require.Equal(t, uintptr(32), unsafe.Sizeof(RawNode{}))
}
