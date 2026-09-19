package glsl

import (
	"context"
	"fmt"
	"sync"

	"github.com/lewtec/lewkit/x/wasm"
	embed "github.com/lewtec/lewkit/x/wasm/glsl/internal/wasm"
)

var (
	compiled *wasm.Compiled
	loadErr  error
	loadOnce sync.Once
)

func load(ctx context.Context) (*wasm.Compiled, error) {
	loadOnce.Do(func() {
		compiled, loadErr = wasm.Compile(ctx, embed.Lib, wasm.Config{Name: "glslang", Emscripten: true})
	})
	return compiled, loadErr
}

func compile(ctx context.Context, src []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mod, err := load(ctx)
	if err != nil {
		return nil, err
	}
	in, err := mod.Instantiate(ctx)
	if err != nil {
		return nil, err
	}
	defer in.Close(ctx)
	srcPtr, err := in.Alloc(ctx, uint32(len(src)+1))
	if err != nil {
		return nil, fmt.Errorf("%w: malloc src", ErrCompile)
	}
	defer in.Free(ctx, srcPtr)
	if err := in.Write(srcPtr, append(append([]byte(nil), src...), 0)); err != nil {
		return nil, fmt.Errorf("%w: write src", ErrCompile)
	}
	outPtr, err := in.Alloc(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("%w: malloc out", ErrCompile)
	}
	defer in.Free(ctx, outPtr)
	lenPtr, err := in.Alloc(ctx, 4)
	if err != nil {
		return nil, fmt.Errorf("%w: malloc len", ErrCompile)
	}
	defer in.Free(ctx, lenPtr)
	code, err := in.Call(ctx, "compile_compute", uint64(srcPtr), uint64(len(src)), uint64(outPtr), uint64(lenPtr))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCompile, err)
	}
	if code != 0 {
		msg := "compile failed"
		if p, e := in.Call(ctx, "last_error"); e == nil && p != 0 {
			if s, e := in.CString(uint32(p)); e == nil && s != "" {
				msg = s
			}
		}
		return nil, fmt.Errorf("%w: %s", ErrCompile, msg)
	}
	spvPtr, err := in.Uint32LE(outPtr)
	if err != nil || spvPtr == 0 {
		return nil, fmt.Errorf("%w: empty output", ErrCompile)
	}
	n, err := in.Uint32LE(lenPtr)
	if err != nil || n < 20 || n%4 != 0 {
		return nil, fmt.Errorf("%w: bad length", ErrCompile)
	}
	raw, err := in.Read(spvPtr, n)
	if err != nil {
		return nil, fmt.Errorf("%w: read spirv", ErrCompile)
	}
	out := append([]byte(nil), raw...)
	in.Free(ctx, spvPtr)
	return out, nil
}
