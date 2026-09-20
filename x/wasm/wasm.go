// Package wasm loads embedded WebAssembly with wazero (WASI, optional Emscripten).
package wasm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// Config is how to compile a module.
type Config struct {
	// Name keys the on-disk compilation cache.
	Name string
	// Emscripten instantiates env (including a getcwd stub).
	Emscripten bool
}

// Compiled is a WASI module ready to instantiate.
type Compiled struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
}

// Compile compiles bin. The runtime is process-wide per Name.
func Compile(ctx context.Context, bin []byte, cfg Config) (*Compiled, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(bin) == 0 {
		return nil, ErrEmpty
	}
	if cfg.Name == "" {
		cfg.Name = "module"
	}
	config := wazero.NewRuntimeConfig()
	if dir, err := os.UserCacheDir(); err == nil {
		cache, err := wazero.NewCompilationCacheWithDir(filepath.Join(dir, "lewtec-lewkit-wasm", cfg.Name))
		if err == nil {
			config = config.WithCompilationCache(cache)
		}
	}
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return nil, errors.Join(fmt.Errorf("%w: %w", ErrWASI, err), runtime.Close(ctx))
	}
	mod, err := runtime.CompileModule(ctx, bin)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("%w: %s: %w", ErrCompile, cfg.Name, err), runtime.Close(ctx))
	}
	if cfg.Emscripten {
		if err := instantiateEmscripten(ctx, runtime, mod); err != nil {
			return nil, errors.Join(err, runtime.Close(ctx))
		}
	}
	return &Compiled{runtime: runtime, module: mod}, nil
}

func instantiateEmscripten(ctx context.Context, runtime wazero.Runtime, mod wazero.CompiledModule) error {
	exporter, err := emscripten.NewFunctionExporterForModule(mod)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrEnv, err)
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
		return fmt.Errorf("%w: %w", ErrEnv, err)
	}
	return nil
}

// Instance is one instantiation of a compiled module.
type Instance struct {
	mod    api.Module
	mem    api.Memory
	malloc api.Function
	free   api.Function
}

// Instantiate creates a guest instance. Call Close when done.
func (c *Compiled) Instantiate(ctx context.Context) (*Instance, error) {
	if c == nil {
		return nil, ErrNil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mod, err := c.runtime.InstantiateModule(ctx, c.module, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInst, err)
	}
	if init := mod.ExportedFunction("_initialize"); init != nil {
		if _, err := init.Call(ctx); err != nil {
			return nil, errors.Join(fmt.Errorf("%w: %w", ErrInit, err), mod.Close(ctx))
		}
	}
	return &Instance{
		mod:    mod,
		mem:    mod.Memory(),
		malloc: mod.ExportedFunction("malloc"),
		free:   mod.ExportedFunction("free"),
	}, nil
}

// Close destroys the instance.
func (in *Instance) Close(ctx context.Context) error {
	if in == nil || in.mod == nil {
		return nil
	}
	err := in.mod.Close(ctx)
	in.mod = nil
	return err
}

// Export returns a guest function, or nil.
func (in *Instance) Export(name string) api.Function {
	if in == nil || in.mod == nil {
		return nil
	}
	return in.mod.ExportedFunction(name)
}

// Call invokes name. Missing export is an error.
func (in *Instance) Call(ctx context.Context, name string, args ...uint64) (uint64, error) {
	fn := in.Export(name)
	if fn == nil {
		return 0, fmt.Errorf("%w: %s", ErrExport, name)
	}
	return Call(ctx, fn, args...)
}

// Alloc is guest malloc.
func (in *Instance) Alloc(ctx context.Context, n uint32) (uint32, error) {
	if n == 0 {
		n = 1
	}
	if in.malloc == nil {
		return 0, ErrMalloc
	}
	p, err := Call(ctx, in.malloc, uint64(n))
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrMalloc, err)
	}
	if p == 0 {
		return 0, fmt.Errorf("%w: oom", ErrMalloc)
	}
	return uint32(p), nil
}

// Free is guest free.
func (in *Instance) Free(ctx context.Context, p uint32) {
	if in == nil || in.free == nil || p == 0 {
		return
	}
	if _, err := Call(ctx, in.free, uint64(p)); err != nil {
		return
	}
}

// Write copies b into guest memory at p.
func (in *Instance) Write(p uint32, b []byte) error {
	if in == nil || in.mem == nil || !in.mem.Write(p, b) {
		return ErrMemory
	}
	return nil
}

// Read copies n bytes from guest memory at p.
func (in *Instance) Read(p, n uint32) ([]byte, error) {
	if in == nil || in.mem == nil {
		return nil, ErrMemory
	}
	b, ok := in.mem.Read(p, n)
	if !ok {
		return nil, ErrMemory
	}
	return b, nil
}

// Uint32LE reads a little-endian uint32.
func (in *Instance) Uint32LE(p uint32) (uint32, error) {
	if in == nil || in.mem == nil {
		return 0, ErrMemory
	}
	v, ok := in.mem.ReadUint32Le(p)
	if !ok {
		return 0, ErrMemory
	}
	return v, nil
}

// CString reads a NUL-terminated string.
func (in *Instance) CString(p uint32) (string, error) {
	if in == nil || in.mem == nil {
		return "", ErrMemory
	}
	s, ok := ReadCString(in.mem, p)
	if !ok {
		return "", ErrMemory
	}
	return s, nil
}

// Call invokes fn and returns the first result, or 0.
func Call(ctx context.Context, fn api.Function, args ...uint64) (uint64, error) {
	results, err := fn.Call(ctx, args...)
	if err != nil {
		return 0, err
	}
	if len(results) == 0 {
		return 0, nil
	}
	return results[0], nil
}

// ReadCString reads a NUL-terminated string from guest memory.
func ReadCString(memory api.Memory, pointer uint32) (string, bool) {
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
