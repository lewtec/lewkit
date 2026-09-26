package jsonleaven

import (
	"unsafe"

	"github.com/lewtec/leaven-tree-sitter/grammar"
)

func init() {
	grammar.Register("json", (*grammar.TSLanguage)(unsafe.Pointer(tree_sitter_json())))
}
