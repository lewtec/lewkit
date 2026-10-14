package wasm

import (
	"errors"

	"github.com/lewtec/lewkit/x/errs"
)

var (
	ErrEmpty   = errs.Wrap(errs.ErrEmpty, "empty wasm")
	ErrNil     = errors.New("nil compiled module")
	ErrWASI    = errors.New("wasi")
	ErrCompile = errors.New("compile wasm")
	ErrEnv     = errors.New("emscripten env")
	ErrInst    = errors.New("instantiate")
	ErrInit    = errors.New("initialize")
	ErrExport  = errs.Wrap(errs.ErrMissing, "missing export")
	ErrMalloc  = errors.New("malloc")
	ErrMemory  = errors.New("wasm memory")
)
