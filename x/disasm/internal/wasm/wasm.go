// Package wasm embeds the Capstone WASI reactor.
//
// The blob is libcapstone built from Capstone
// f6ab2ab9152e94687f112cda9d0cb5c0745ff059 with wasi-sdk-21,
// plus cs_get_mnemonic / cs_get_op_str from wasilibs/go-capstone.
// Rebuild with that repository's buildtools/wasm/Dockerfile.
package wasm

import _ "embed"

//go:embed libcapstone.wasm
var Lib []byte
