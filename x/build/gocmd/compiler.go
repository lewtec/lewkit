package gocmd

import (
	"context"
)

// Compiler selects the go command or the embedded compiler.
// The zero value runs the host go command.
// CompilerLewkit runs only when x/build/gocmd/lewkit is linked.
//
//lewkit:enum
type Compiler int

const (
	// CompilerHost runs the go command.
	CompilerHost Compiler = iota + 1
	// CompilerLewkit compiles a CGO-free amd64 or arm64 binary with the embedded compiler.
	// The compiler is linked by importing x/build/gocmd/lewkit.
	CompilerLewkit
)

type compilerKey struct{}

// WithCompiler returns ctx that carries the compiler release builds use.
func WithCompiler(ctx context.Context, c Compiler) context.Context {
	return context.WithValue(ctx, compilerKey{}, c)
}

func (c Command) active(ctx context.Context) Compiler {
	if c.Compiler != 0 {
		return c.Compiler
	}
	got, _ := ctx.Value(compilerKey{}).(Compiler)
	return got
}

// runLewkit is set by x/build/gocmd/lewkit. Nil means that package is not linked.
var runLewkit func(context.Context, Command) error

// SetLewkit registers the embedded compiler.
// x/build/gocmd/lewkit calls this from init.
func SetLewkit(fn func(context.Context, Command) error) {
	runLewkit = fn
}
