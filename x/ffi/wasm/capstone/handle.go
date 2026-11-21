package capstone

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"iter"

	"github.com/lewtec/lewkit/x/ffi/wasm"
	embed "github.com/lewtec/lewkit/x/ffi/wasm/capstone/internal/wasm"
	"github.com/lewtec/lewkit/x/singleton"
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

var (
	ErrClosed = errors.New("capstone closed")
	ErrOpen   = errors.New("cs_open")
	ErrOption = errors.New("cs_option")
	ErrMalloc = errors.New("cs_malloc")
)

// Instruction is one decoded instruction.
type Instruction struct {
	ID       uint32
	Address  uint64
	Size     uint16
	Bytes    []byte
	Mnemonic string
	Operands string
}

// Handle is one Capstone engine inside a wasm instance.
type Handle struct {
	in     *wasm.Instance
	handle uint32
}

var compiled = singleton.NewSingleton(func(ctx context.Context) (*wasm.Compiled, error) {
	return wasm.Compile(context.WithoutCancel(ctx), embed.Lib, wasm.Config{Name: "capstone"})
})

var requiredExports = []string{
	"malloc",
	"free",
	"cs_open",
	"cs_close",
	"cs_option",
	"cs_disasm_iter",
	"cs_malloc",
	"cs_free",
	"cs_get_mnemonic",
	"cs_get_op_str",
}

// Open loads Capstone and calls cs_open.
func Open(ctx context.Context, architecture, mode uint32) (*Handle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	mod, err := compiled.GetContext(ctx)
	if err != nil {
		return nil, err
	}
	in, err := mod.Instantiate(ctx)
	if err != nil {
		return nil, err
	}
	for _, name := range requiredExports {
		if in.Export(name) == nil {
			return nil, closeInstance(ctx, in, fmt.Errorf("%w: %s", wasm.ErrExport, name))
		}
	}
	opened := &Handle{in: in}
	handlePointer, err := opened.alloc(ctx, 4)
	if err != nil {
		return nil, closeInstance(ctx, in, err)
	}
	code, err := in.Call(ctx, "cs_open", uint64(architecture), uint64(mode), uint64(handlePointer))
	if err != nil {
		opened.dealloc(ctx, handlePointer)
		return nil, closeInstance(ctx, in, fmt.Errorf("cs_open: %w", err))
	}
	if code != 0 {
		message := opened.strerror(ctx, uint32(code))
		opened.dealloc(ctx, handlePointer)
		return nil, closeInstance(ctx, in, fmt.Errorf("%w: %s", ErrOpen, message))
	}
	raw, err := in.Uint32LE(handlePointer)
	opened.dealloc(ctx, handlePointer)
	if err != nil {
		return nil, closeInstance(ctx, in, fmt.Errorf("cs_open: %w", err))
	}
	if raw == 0 {
		return nil, closeInstance(ctx, in, fmt.Errorf("%w: empty handle", ErrOpen))
	}
	opened.handle = raw
	return opened, nil
}

// Syntax sets CS_OPT_SYNTAX.
func (handle *Handle) Syntax(ctx context.Context, syntax uint32) error {
	return handle.setOption(ctx, optionSyntax, syntax)
}

// SkipData sets CS_OPT_SKIPDATA.
func (handle *Handle) SkipData(ctx context.Context, enabled bool) error {
	value := uint32(0)
	if enabled {
		value = optionOn
	}
	return handle.setOption(ctx, optionSkipData, value)
}

// Close releases the engine and the wasm instance.
func (handle *Handle) Close(ctx context.Context) error {
	if handle == nil || handle.in == nil {
		return nil
	}
	if handle.handle != 0 {
		if pointer, err := handle.alloc(ctx, 4); err == nil {
			if err := handle.writeUint32LE(pointer, handle.handle); err == nil {
				_, _ = handle.in.Call(ctx, "cs_close", uint64(pointer))
			}
			handle.dealloc(ctx, pointer)
		}
		handle.handle = 0
	}
	err := handle.in.Close(ctx)
	handle.in = nil
	return err
}

// Iter yields instructions from code starting at address.
func (handle *Handle) Iter(ctx context.Context, code []byte, address uint64) iter.Seq2[Instruction, error] {
	return func(yield func(Instruction, error) bool) {
		if handle == nil || handle.in == nil {
			yield(Instruction{}, ErrClosed)
			return
		}
		if err := handle.walk(ctx, code, address, yield); err != nil {
			yield(Instruction{}, err)
		}
	}
}

func (handle *Handle) setOption(ctx context.Context, kind, value uint32) error {
	code, err := handle.in.Call(ctx, "cs_option", uint64(handle.handle), uint64(kind), uint64(value))
	if err != nil {
		return fmt.Errorf("cs_option: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("%w: %s", ErrOption, handle.strerror(ctx, uint32(code)))
	}
	return nil
}

func (handle *Handle) walk(ctx context.Context, code []byte, address uint64, yield func(Instruction, error) bool) error {
	instructionPointer, err := handle.in.Call(ctx, "cs_malloc", uint64(handle.handle))
	if err != nil {
		return fmt.Errorf("cs_malloc: %w", err)
	}
	if instructionPointer == 0 {
		return fmt.Errorf("%w: empty instruction", ErrMalloc)
	}
	defer func() { _, _ = handle.in.Call(ctx, "cs_free", instructionPointer, 1) }()

	metaPointer, err := handle.alloc(ctx, 16)
	if err != nil {
		return err
	}
	defer handle.dealloc(ctx, metaPointer)
	codePointerPointer := metaPointer
	sizePointer := metaPointer + 4
	addressPointer := metaPointer + 8

	var codePointer uint32
	if len(code) > 0 {
		codePointer, err = handle.alloc(ctx, uint32(len(code)))
		if err != nil {
			return err
		}
		defer handle.dealloc(ctx, codePointer)
		if err := handle.in.Write(codePointer, code); err != nil {
			return fmt.Errorf("write code: %w", err)
		}
	}
	if err := handle.writeUint32LE(codePointerPointer, codePointer); err != nil {
		return fmt.Errorf("write code pointer: %w", err)
	}
	if err := handle.writeUint32LE(sizePointer, uint32(len(code))); err != nil {
		return fmt.Errorf("write code size: %w", err)
	}
	if err := handle.writeUint64LE(addressPointer, address); err != nil {
		return fmt.Errorf("write address: %w", err)
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ok, err := handle.in.Call(ctx, "cs_disasm_iter", uint64(handle.handle), uint64(codePointerPointer), uint64(sizePointer), uint64(addressPointer), instructionPointer)
		if err != nil {
			return fmt.Errorf("cs_disasm_iter: %w", err)
		}
		if ok == 0 {
			return nil
		}
		instruction, err := handle.readInstruction(ctx, uint32(instructionPointer))
		if err != nil {
			return err
		}
		if !yield(instruction, nil) {
			return nil
		}
	}
}

func (handle *Handle) alloc(ctx context.Context, size uint32) (uint32, error) {
	return handle.in.Alloc(ctx, size)
}

func (handle *Handle) dealloc(ctx context.Context, pointer uint32) {
	handle.in.Free(ctx, pointer)
}

func (handle *Handle) writeUint32LE(pointer, value uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], value)
	return handle.in.Write(pointer, buf[:])
}

func (handle *Handle) writeUint64LE(pointer uint32, value uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], value)
	return handle.in.Write(pointer, buf[:])
}

func (handle *Handle) uint16LE(pointer uint32) (uint16, error) {
	buf, err := handle.in.Read(pointer, 2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buf), nil
}

func (handle *Handle) uint64LE(pointer uint32) (uint64, error) {
	buf, err := handle.in.Read(pointer, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func (handle *Handle) strerror(ctx context.Context, code uint32) string {
	pointer, err := handle.in.Call(ctx, "cs_strerror", uint64(code))
	if err != nil || pointer == 0 {
		return fmt.Sprintf("capstone error %d", code)
	}
	message, err := handle.in.CString(uint32(pointer))
	if err != nil || message == "" {
		return fmt.Sprintf("capstone error %d", code)
	}
	return message
}

func (handle *Handle) readInstruction(ctx context.Context, instructionPointer uint32) (Instruction, error) {
	id, err := handle.in.Uint32LE(instructionPointer + instructionOffsetID)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction id: %w", err)
	}
	address, err := handle.uint64LE(instructionPointer + instructionOffsetAddr)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction address: %w", err)
	}
	size, err := handle.uint16LE(instructionPointer + instructionOffsetSize)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction size: %w", err)
	}
	raw, err := handle.in.Read(instructionPointer+instructionOffsetBytes, instructionMaxBytes)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction bytes: %w", err)
	}
	if int(size) > len(raw) {
		size = uint16(len(raw))
	}
	mnemonicPointer, err := handle.in.Call(ctx, "cs_get_mnemonic", uint64(instructionPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("mnemonic: %w", err)
	}
	operandsPointer, err := handle.in.Call(ctx, "cs_get_op_str", uint64(instructionPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("operands: %w", err)
	}
	mnemonic, err := handle.in.CString(uint32(mnemonicPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("read mnemonic: %w", err)
	}
	operands, err := handle.in.CString(uint32(operandsPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("read operands: %w", err)
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

func closeInstance(ctx context.Context, in *wasm.Instance, err error) error {
	return errors.Join(err, in.Close(ctx))
}
