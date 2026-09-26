package jsonwazero

import (
	_ "embed"

	"github.com/lewtec/wazero-tree-sitter/grammar"
)

//go:embed grammar.wasm
var wasm []byte

func init() {
	grammar.Register("json", grammar.NewLanguage(wasm))
}
