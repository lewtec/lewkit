package disasm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"iter"
	"sync"
)

var _ io.Closer = (*Engine)(nil)

// Instruction is one decoded instruction.
type Instruction struct {
	ID       uint32
	Address  uint64
	Size     uint16
	Bytes    []byte
	Mnemonic string
	Operands string
}

type engineOptions struct {
	syntax   Syntax
	skipData bool
}

// Option configures [Open].
type Option func(*engineOptions)

// WithSyntax sets the assembly syntax.
func WithSyntax(syntax Syntax) Option {
	return func(options *engineOptions) { options.syntax = syntax }
}

// WithSkipData skips bytes that do not decode as instructions.
func WithSkipData(enabled bool) Option {
	return func(options *engineOptions) { options.skipData = enabled }
}

// Engine is an open Capstone handle.
type Engine struct {
	mu      sync.Mutex
	session *session
}

// Open creates an engine for architecture and mode.
func Open(ctx context.Context, architecture Architecture, mode Mode, opts ...Option) (*Engine, error) {
	options := engineOptions{syntax: SyntaxDefault}
	for _, opt := range opts {
		opt(&options)
	}
	session, err := openSession(ctx, architecture, mode)
	if err != nil {
		return nil, err
	}
	engine := &Engine{session: session}
	if options.syntax != 0 {
		if err := session.setOption(ctx, optionSyntax, uint32(options.syntax)); err != nil {
			return nil, errors.Join(err, engine.Close())
		}
	}
	if options.skipData {
		if err := session.setOption(ctx, optionSkipData, optionOn); err != nil {
			return nil, errors.Join(err, engine.Close())
		}
	}
	return engine, nil
}

// Close releases the engine.
func (engine *Engine) Close() error {
	if engine == nil {
		return nil
	}
	engine.mu.Lock()
	defer engine.mu.Unlock()
	err := engine.session.close(context.Background())
	engine.session = nil
	return err
}

// Disassemble decodes code starting at address.
func (engine *Engine) Disassemble(ctx context.Context, code []byte, address uint64) ([]Instruction, error) {
	var instructions []Instruction
	for instruction, err := range engine.Iter(ctx, code, address) {
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, instruction)
	}
	return instructions, nil
}

// Iter yields instructions from code starting at address.
func (engine *Engine) Iter(ctx context.Context, code []byte, address uint64) iter.Seq2[Instruction, error] {
	return func(yield func(Instruction, error) bool) {
		engine.mu.Lock()
		defer engine.mu.Unlock()
		if engine.session == nil {
			yield(Instruction{}, fmt.Errorf("engine closed"))
			return
		}
		if err := engine.walk(ctx, code, address, yield); err != nil {
			yield(Instruction{}, err)
		}
	}
}

func (engine *Engine) walk(ctx context.Context, code []byte, address uint64, yield func(Instruction, error) bool) error {
	session := engine.session
	instructionPointer, err := call(ctx, session.allocateInstruction, uint64(session.handle))
	if err != nil {
		return fmt.Errorf("cs_malloc: %w", err)
	}
	if instructionPointer == 0 {
		return fmt.Errorf("cs_malloc: empty instruction")
	}
	defer func() { _, _ = call(ctx, session.freeInstruction, instructionPointer, 1) }()

	metaPointer, err := session.alloc(ctx, 16)
	if err != nil {
		return err
	}
	defer session.dealloc(ctx, metaPointer)
	codePointerPointer := metaPointer
	sizePointer := metaPointer + 4
	addressPointer := metaPointer + 8

	var codePointer uint32
	if len(code) > 0 {
		codePointer, err = session.alloc(ctx, uint32(len(code)))
		if err != nil {
			return err
		}
		defer session.dealloc(ctx, codePointer)
		if !session.memory.Write(codePointer, code) {
			return fmt.Errorf("write code")
		}
	}
	if !session.memory.WriteUint32Le(codePointerPointer, codePointer) {
		return fmt.Errorf("write code pointer")
	}
	if !session.memory.WriteUint32Le(sizePointer, uint32(len(code))) {
		return fmt.Errorf("write code size")
	}
	if !session.memory.WriteUint64Le(addressPointer, address) {
		return fmt.Errorf("write address")
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		ok, err := call(ctx, session.disassembleIter, uint64(session.handle), uint64(codePointerPointer), uint64(sizePointer), uint64(addressPointer), instructionPointer)
		if err != nil {
			return fmt.Errorf("cs_disasm_iter: %w", err)
		}
		if ok == 0 {
			return nil
		}
		instruction, err := session.readInstruction(ctx, uint32(instructionPointer))
		if err != nil {
			return err
		}
		if !yield(instruction, nil) {
			return nil
		}
	}
}
