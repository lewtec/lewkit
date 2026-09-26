//go:build (linux && (386 || amd64 || arm64)) || (darwin && (amd64 || arm64)) || (windows && (amd64 || arm64))

package jsonccgo

import (
	"unsafe"

	"github.com/modernc-tree-sitter/ccgo-tree-sitter/grammar"
)

func init() {
	grammar.Register("json", (*grammar.TSLanguage)(unsafe.Pointer(tree_sitter_json(nil))))
}
