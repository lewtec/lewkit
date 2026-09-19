package wasm

import "errors"

var (
	ErrEmpty   = errors.New("empty wasm")
	ErrNil     = errors.New("nil compiled module")
	ErrWASI    = errors.New("wasi")
	ErrCompile = errors.New("compile wasm")
	ErrEnv     = errors.New("emscripten env")
	ErrInst    = errors.New("instantiate")
	ErrInit    = errors.New("initialize")
	ErrExport  = errors.New("missing export")
	ErrMalloc  = errors.New("malloc")
	ErrMemory  = errors.New("wasm memory")
)
