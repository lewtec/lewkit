package glsl

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/lewtec/lewkit/x/wasm/glsl/internal/wasm"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

type compiled struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
}

var load = sync.OnceValues(func() (compiled, error) {
	ctx := context.Background()
	config := wazero.NewRuntimeConfig()
	if dir, err := os.UserCacheDir(); err == nil {
		cache, err := wazero.NewCompilationCacheWithDir(filepath.Join(dir, "lewtec-lewkit-glsl"))
		if err == nil {
			config = config.WithCompilationCache(cache)
		}
	}
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return compiled{}, errors.Join(fmt.Errorf("wasi: %w", err), runtime.Close(ctx))
	}
	mod, err := runtime.CompileModule(ctx, wasm.Lib)
	if err != nil {
		return compiled{}, errors.Join(fmt.Errorf("compile glslang: %w", err), runtime.Close(ctx))
	}
	exporter, err := emscripten.NewFunctionExporterForModule(mod)
	if err != nil {
		return compiled{}, errors.Join(fmt.Errorf("emscripten: %w", err), runtime.Close(ctx))
	}
	env := runtime.NewHostModuleBuilder("env")
	exporter.ExportFunctions(env)
	env.NewFunctionBuilder().
		WithFunc(func(_ context.Context, m api.Module, buf, size int32) int32 {
			if size < 2 {
				return -1
			}
			if !m.Memory().Write(uint32(buf), []byte{'/', 0}) {
				return -1
			}
			return 1
		}).
		Export("__syscall_getcwd")
	if _, err := env.Instantiate(ctx); err != nil {
		return compiled{}, errors.Join(fmt.Errorf("env: %w", err), runtime.Close(ctx))
	}
	return compiled{runtime: runtime, module: mod}, nil
})

func compile(ctx context.Context, src []byte) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c, err := load()
	if err != nil {
		return nil, err
	}
	mod, err := c.runtime.InstantiateModule(ctx, c.module, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		return nil, fmt.Errorf("instantiate glslang: %w", err)
	}
	defer mod.Close(ctx)
	if init := mod.ExportedFunction("_initialize"); init != nil {
		if _, err := init.Call(ctx); err != nil {
			return nil, fmt.Errorf("initialize glslang: %w", err)
		}
	}
	mem := mod.Memory()
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	fn := mod.ExportedFunction("compile_compute")
	errFn := mod.ExportedFunction("last_error")
	if malloc == nil || free == nil || fn == nil {
		return nil, fmt.Errorf("%w: glslang exports missing", ErrCompile)
	}
	srcPtr, err := call(ctx, malloc, uint64(len(src)+1))
	if err != nil || srcPtr == 0 {
		return nil, fmt.Errorf("%w: malloc src", ErrCompile)
	}
	defer func() { _, _ = call(ctx, free, srcPtr) }()
	if !mem.Write(uint32(srcPtr), append(append([]byte(nil), src...), 0)) {
		return nil, fmt.Errorf("%w: write src", ErrCompile)
	}
	outPtr, err := call(ctx, malloc, 4)
	if err != nil || outPtr == 0 {
		return nil, fmt.Errorf("%w: malloc out", ErrCompile)
	}
	defer func() { _, _ = call(ctx, free, outPtr) }()
	lenPtr, err := call(ctx, malloc, 4)
	if err != nil || lenPtr == 0 {
		return nil, fmt.Errorf("%w: malloc len", ErrCompile)
	}
	defer func() { _, _ = call(ctx, free, lenPtr) }()
	code, err := call(ctx, fn, srcPtr, uint64(len(src)), outPtr, lenPtr)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCompile, err)
	}
	if code != 0 {
		msg := "compile failed"
		if errFn != nil {
			if p, e := call(ctx, errFn); e == nil && p != 0 {
				if s, ok := readCString(mem, uint32(p)); ok && s != "" {
					msg = s
				}
			}
		}
		return nil, fmt.Errorf("%w: %s", ErrCompile, msg)
	}
	spvPtr, ok := mem.ReadUint32Le(uint32(outPtr))
	if !ok || spvPtr == 0 {
		return nil, fmt.Errorf("%w: empty output", ErrCompile)
	}
	n, ok := mem.ReadUint32Le(uint32(lenPtr))
	if !ok || n < 20 || n%4 != 0 {
		return nil, fmt.Errorf("%w: bad length", ErrCompile)
	}
	raw, ok := mem.Read(spvPtr, n)
	if !ok {
		return nil, fmt.Errorf("%w: read spirv", ErrCompile)
	}
	out := append([]byte(nil), raw...)
	_, _ = call(ctx, free, uint64(spvPtr))
	return out, nil
}

func call(ctx context.Context, fn api.Function, args ...uint64) (uint64, error) {
	results, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0], nil
}

func readCString(memory api.Memory, pointer uint32) (string, bool) {
	if pointer == 0 {
		return "", true
	}
	end := pointer
	for {
		b, ok := memory.ReadByte(end)
		if !ok {
			return "", false
		}
		if b == 0 {
			break
		}
		end++
	}
	if end == pointer {
		return "", true
	}
	buf, ok := memory.Read(pointer, end-pointer)
	if !ok {
		return "", false
	}
	return string(buf), true
}
