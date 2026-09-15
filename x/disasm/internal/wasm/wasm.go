// Package wasm embeds the Capstone WASI reactor.
//
// libcapstone.wasm is placed from github:wasilibs/go-capstone
// (input capstone_wasm in workspaced.cue). Refresh with go generate.
package wasm

import _ "embed"

//go:embed libcapstone.wasm
var Lib []byte
