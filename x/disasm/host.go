package disasm

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"

	embed "github.com/lewtec/lewkit/x/disasm/internal/wasm"
	"github.com/lewtec/lewkit/x/wasm"
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

var load = sync.OnceValues(func() (*wasm.Compiled, error) {
	return wasm.Compile(context.Background(), embed.Lib, wasm.Config{Name: "capstone"})
})

type session struct {
	in     *wasm.Instance
	handle uint32
}

func loadCompiled(ctx context.Context) (*wasm.Compiled, error) {
	done := make(chan struct {
		c   *wasm.Compiled
		err error
	}, 1)
	go func() {
		c, err := load()
		done <- struct {
			c   *wasm.Compiled
			err error
		}{c, err}
	}()
	select {
	case <-ctx.Done():
		return nil, context.Cause(ctx)
	case r := <-done:
		return r.c, r.err
	}
}

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

func openSession(ctx context.Context, architecture Architecture, mode Mode) (*session, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	compiled, err := loadCompiled(ctx)
	if err != nil {
		return nil, err
	}
	in, err := compiled.Instantiate(ctx)
	if err != nil {
		return nil, err
	}
	for _, name := range requiredExports {
		if in.Export(name) == nil {
			return nil, closeInstance(ctx, in, fmt.Errorf("%w: %s", wasm.ErrExport, name))
		}
	}
	opened := &session{in: in}
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
		return nil, closeInstance(ctx, in, fmt.Errorf("cs_open: %s", message))
	}
	handle, err := in.Uint32LE(handlePointer)
	opened.dealloc(ctx, handlePointer)
	if err != nil {
		return nil, closeInstance(ctx, in, fmt.Errorf("cs_open: %w", err))
	}
	if handle == 0 {
		return nil, closeInstance(ctx, in, fmt.Errorf("cs_open: empty handle"))
	}
	opened.handle = handle
	return opened, nil
}

func (session *session) setOption(ctx context.Context, typ, value uint32) error {
	code, err := session.in.Call(ctx, "cs_option", uint64(session.handle), uint64(typ), uint64(value))
	if err != nil {
		return fmt.Errorf("cs_option: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("cs_option: %s", session.strerror(ctx, uint32(code)))
	}
	return nil
}

func (session *session) close(ctx context.Context) error {
	if session == nil || session.in == nil {
		return nil
	}
	if session.handle != 0 {
		if pointer, err := session.alloc(ctx, 4); err == nil {
			if err := session.writeUint32LE(pointer, session.handle); err == nil {
				_, _ = session.in.Call(ctx, "cs_close", uint64(pointer))
			}
			session.dealloc(ctx, pointer)
		}
		session.handle = 0
	}
	err := session.in.Close(ctx)
	session.in = nil
	return err
}

func (session *session) alloc(ctx context.Context, size uint32) (uint32, error) {
	return session.in.Alloc(ctx, size)
}

func (session *session) dealloc(ctx context.Context, pointer uint32) {
	session.in.Free(ctx, pointer)
}

func (session *session) writeUint32LE(pointer, value uint32) error {
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], value)
	return session.in.Write(pointer, buf[:])
}

func (session *session) writeUint64LE(pointer uint32, value uint64) error {
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], value)
	return session.in.Write(pointer, buf[:])
}

func (session *session) uint16LE(pointer uint32) (uint16, error) {
	buf, err := session.in.Read(pointer, 2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(buf), nil
}

func (session *session) uint64LE(pointer uint32) (uint64, error) {
	buf, err := session.in.Read(pointer, 8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(buf), nil
}

func (session *session) strerror(ctx context.Context, code uint32) string {
	pointer, err := session.in.Call(ctx, "cs_strerror", uint64(code))
	if err != nil || pointer == 0 {
		return fmt.Sprintf("capstone error %d", code)
	}
	message, err := session.in.CString(uint32(pointer))
	if err != nil || message == "" {
		return fmt.Sprintf("capstone error %d", code)
	}
	return message
}

func (session *session) readInstruction(ctx context.Context, instructionPointer uint32) (Instruction, error) {
	id, err := session.in.Uint32LE(instructionPointer + instructionOffsetID)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction id: %w", err)
	}
	address, err := session.uint64LE(instructionPointer + instructionOffsetAddr)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction address: %w", err)
	}
	size, err := session.uint16LE(instructionPointer + instructionOffsetSize)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction size: %w", err)
	}
	raw, err := session.in.Read(instructionPointer+instructionOffsetBytes, instructionMaxBytes)
	if err != nil {
		return Instruction{}, fmt.Errorf("read instruction bytes: %w", err)
	}
	if int(size) > len(raw) {
		size = uint16(len(raw))
	}
	mnemonicPointer, err := session.in.Call(ctx, "cs_get_mnemonic", uint64(instructionPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("mnemonic: %w", err)
	}
	operandsPointer, err := session.in.Call(ctx, "cs_get_op_str", uint64(instructionPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("operands: %w", err)
	}
	mnemonic, err := session.in.CString(uint32(mnemonicPointer))
	if err != nil {
		return Instruction{}, fmt.Errorf("read mnemonic: %w", err)
	}
	operands, err := session.in.CString(uint32(operandsPointer))
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
