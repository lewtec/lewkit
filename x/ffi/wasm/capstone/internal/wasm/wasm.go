// Package wasm embeds the Capstone WASI reactor.
//
// libcapstone.wasm is wasilibs/go-capstone @ c90245dc26ed (git main
// when locked). That tree builds Capstone f6ab2ab with wasi-sdk-21.
// Refresh with go generate.
package wasm

import _ "embed"

//go:embed libcapstone.wasm
var Lib []byte
