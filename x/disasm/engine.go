package disasm

import (
	"context"
	"errors"
	"io"
	"iter"
	"sync"

	"github.com/lewtec/lewkit/x/ffi/wasm/capstone"
)

var _ io.Closer = (*Engine)(nil)

var errEngineClosed = errors.New("engine closed")

// Instruction is one decoded instruction.
type Instruction = capstone.Instruction

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
	mu     sync.Mutex
	handle *capstone.Handle
	ctx    context.Context
}

// Open creates an engine for architecture and mode.
func Open(ctx context.Context, architecture Architecture, mode Mode, opts ...Option) (*Engine, error) {
	options := engineOptions{syntax: SyntaxDefault}
	for _, opt := range opts {
		opt(&options)
	}
	handle, err := capstone.Open(ctx, uint32(architecture), uint32(mode))
	if err != nil {
		return nil, err
	}
	engine := &Engine{handle: handle, ctx: context.WithoutCancel(ctx)}
	if options.syntax != 0 && options.syntax != SyntaxDefault {
		if err := handle.Syntax(ctx, uint32(options.syntax)); err != nil {
			return nil, errors.Join(err, engine.Close())
		}
	}
	if options.skipData {
		if err := handle.SkipData(ctx, true); err != nil {
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
	err := engine.handle.Close(engine.ctx)
	engine.handle = nil
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
		if engine.handle == nil {
			yield(Instruction{}, errEngineClosed)
			return
		}
		for instruction, err := range engine.handle.Iter(ctx, code, address) {
			if !yield(instruction, err) {
				return
			}
		}
	}
}
