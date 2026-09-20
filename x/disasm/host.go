package disasm

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	embed "github.com/lewtec/lewkit/x/disasm/internal/wasm"
	"github.com/lewtec/lewkit/x/wasm"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	optionSyntax   uint32 = 1
	optionSkipData uint32 = 5
	optionOn       uint32 = 1

	// cs_insn on wasm32 with 8-byte alignment (id, alias_id, address, size, bytes).
	instructionOffsetID    = 0
	instructionOffsetAddr  = 16
	instructionOffsetSize  = 24
	instructionOffsetBytes = 26
	instructionMaxBytes    = 24
)

type compiled struct {
	runtime wazero.Runtime
	module  wazero.CompiledModule
}

var load = sync.OnceValues(func() (compiled, error) {
	ctx := context.Background()
	config := wazero.NewRuntimeConfig()
	if dir, err := os.UserCacheDir(); err == nil {
		cache, err := wazero.NewCompilationCacheWithDir(filepath.Join(dir, "lewtec-lewkit-disasm"))
		if err == nil {
			config = config.WithCompilationCache(cache)
		}
	}
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return compiled{}, errors.Join(fmt.Errorf("wasi: %w", err), runtime.Close(ctx))
	}
	module, err := runtime.CompileModule(ctx, embed.Lib)
	if err != nil {
		return compiled{}, errors.Join(fmt.Errorf("compile capstone: %w", err), runtime.Close(ctx))
	}
	return compiled{runtime: runtime, module: module}, nil
})

type session struct {
	module              api.Module
	memory              api.Memory
	handle              uint32
	malloc              api.Function
	free                api.Function
	closeHandle         api.Function
	option              api.Function
	disassembleIter     api.Function
	allocateInstruction api.Function
	freeInstruction     api.Function
	mnemonic            api.Function
	operands            api.Function
	errorString         api.Function
}

func loadCompiled(ctx context.Context) (compiled, error) {
	done := make(chan struct {
		c   compiled
		err error
	}, 1)
	go func() {
		c, err := load()
		done <- struct {
			c   compiled
			err error
		}{c, err}
	}()
	select {
	case <-ctx.Done():
		return compiled{}, context.Cause(ctx)
	case r := <-done:
		return r.c, r.err
	}
}

func openSession(ctx context.Context, architecture Architecture, mode Mode) (*session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	compiledModule, err := loadCompiled(ctx)
	if err != nil {
		return nil, err
	}
	module, err := compiledModule.runtime.InstantiateModule(ctx, compiledModule.module, wazero.NewModuleConfig().WithStartFunctions())
	if err != nil {
		return nil, fmt.Errorf("instantiate capstone: %w", err)
	}
	opened := &session{
		module:              module,
		memory:              module.Memory(),
		malloc:              module.ExportedFunction("malloc"),
		free:                module.ExportedFunction("free"),
		closeHandle:         module.ExportedFunction("cs_close"),
		option:              module.ExportedFunction("cs_option"),
		disassembleIter:     module.ExportedFunction("cs_disasm_iter"),
		allocateInstruction: module.ExportedFunction("cs_malloc"),
		freeInstruction:     module.ExportedFunction("cs_free"),
		mnemonic:            module.ExportedFunction("cs_get_mnemonic"),
		operands:            module.ExportedFunction("cs_get_op_str"),
		errorString:         module.ExportedFunction("cs_strerror"),
	}
	if opened.malloc == nil || opened.free == nil || opened.closeHandle == nil || opened.option == nil ||
		opened.disassembleIter == nil || opened.allocateInstruction == nil || opened.freeInstruction == nil ||
		opened.mnemonic == nil || opened.operands == nil {
		return nil, closeModule(ctx, module, fmt.Errorf("capstone exports are incomplete"))
	}
	open := module.ExportedFunction("cs_open")
	if open == nil {
		return nil, closeModule(ctx, module, fmt.Errorf("capstone exports are incomplete"))
	}
	handlePointer, err := opened.alloc(ctx, 4)
	if err != nil {
		return nil, closeModule(ctx, module, err)
	}
	code, err := call(ctx, open, uint64(architecture), uint64(mode), uint64(handlePointer))
	if err != nil {
		opened.dealloc(ctx, handlePointer)
		return nil, closeModule(ctx, module, fmt.Errorf("cs_open: %w", err))
	}
	if code != 0 {
		message := opened.strerror(ctx, uint32(code))
		opened.dealloc(ctx, handlePointer)
		return nil, closeModule(ctx, module, fmt.Errorf("cs_open: %s", message))
	}
	handle, ok := opened.memory.ReadUint32Le(handlePointer)
	opened.dealloc(ctx, handlePointer)
	if !ok || handle == 0 {
		return nil, closeModule(ctx, module, fmt.Errorf("cs_open: empty handle"))
	}
	opened.handle = handle
	return opened, nil
}

func (session *session) setOption(ctx context.Context, typ, value uint32) error {
	code, err := call(ctx, session.option, uint64(session.handle), uint64(typ), uint64(value))
	if err != nil {
		return fmt.Errorf("cs_option: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("cs_option: %s", session.strerror(ctx, uint32(code)))
	}
	return nil
}

func (session *session) close(ctx context.Context) error {
	if session == nil || session.module == nil {
		return nil
	}
	if session.handle != 0 && session.closeHandle != nil {
		if pointer, err := session.alloc(ctx, 4); err == nil {
			_ = session.memory.WriteUint32Le(pointer, session.handle)
			_, _ = call(ctx, session.closeHandle, uint64(pointer))
			session.dealloc(ctx, pointer)
		}
		session.handle = 0
	}
	err := session.module.Close(ctx)
	session.module = nil
	return err
}

func (session *session) alloc(ctx context.Context, size uint32) (uint32, error) {
	if size == 0 {
		size = 1
	}
	pointer, err := call(ctx, session.malloc, uint64(size))
	if err != nil {
		return 0, fmt.Errorf("malloc: %w", err)
	}
	if pointer == 0 {
		return 0, fmt.Errorf("malloc: out of memory")
	}
	return uint32(pointer), nil
}

func (session *session) dealloc(ctx context.Context, pointer uint32) {
	if pointer == 0 {
		return
	}
	_, _ = call(ctx, session.free, uint64(pointer))
}

func (session *session) strerror(ctx context.Context, code uint32) string {
	if session.errorString == nil {
		return fmt.Sprintf("capstone error %d", code)
	}
	pointer, err := call(ctx, session.errorString, uint64(code))
	if err != nil || pointer == 0 {
		return fmt.Sprintf("capstone error %d", code)
	}
	message, ok := readCString(session.memory, uint32(pointer))
	if !ok || message == "" {
		return fmt.Sprintf("capstone error %d", code)
	}
	return message
}

func (session *session) readInstruction(ctx context.Context, instructionPointer uint32) (Instruction, error) {
	id, ok := session.memory.ReadUint32Le(instructionPointer + instructionOffsetID)
	if !ok {
		return Instruction{}, fmt.Errorf("read instruction id")
	}
	address, ok := session.memory.ReadUint64Le(instructionPointer + instructionOffsetAddr)
	if !ok {
		return Instruction{}, fmt.Errorf("read instruction address")
	}
	size, ok := session.memory.ReadUint16Le(instructionPointer + instructionOffsetSize)
	if !ok {
		return Instruction{}, fmt.Errorf("read instruction size")
	}
	raw, ok := session.memory.Read(instructionPointer+instructionOffsetBytes, instructionMaxBytes)
	if !ok {
		return Instruction{}, fmt.Errorf("read instruction bytes")
	}
	if int(size) > len(raw) {
		size = uint16(len(raw))
	}
	mnemonicPointer, err := call(ctx, session.mnemonic, uint64(instructionPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("mnemonic: %w", err)
	}
	operandsPointer, err := call(ctx, session.operands, uint64(instructionPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("operands: %w", err)
	}
	mnemonic, ok := readCString(session.memory, uint32(mnemonicPointer))
	if !ok {
		return Instruction{}, fmt.Errorf("read mnemonic")
	}
	operands, ok := readCString(session.memory, uint32(operandsPointer))
	if !ok {
		return Instruction{}, fmt.Errorf("read operands")
	}
	return Instruction{
		ID:       id,
		Address:  address,
		Size:     size,
		Bytes:    append([]byte(nil), raw[:size]...),
		Mnemonic: mnemonic,
		Operands: operands,
	}, nil
}

func closeModule(ctx context.Context, module api.Module, err error) error {
	return errors.Join(err, module.Close(ctx))
}

func call(ctx context.Context, function api.Function, args ...uint64) (uint64, error) {
	return wasm.Call(ctx, function, args...)
}

func readCString(memory api.Memory, pointer uint32) (string, bool) {
	return wasm.ReadCString(memory, pointer)
}
