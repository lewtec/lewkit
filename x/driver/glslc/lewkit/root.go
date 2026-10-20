// Package lewkit compiles compute GLSL with the embedded glslang reactor.
package lewkit

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/glslc"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl/reactor"
)

type factory struct{}

func (factory) ID() string   { return "glslc_lewkit" }
func (factory) Name() string { return "lewkit" }
func (factory) Weight() int  { return 40 }

func (factory) CheckCompatibility(context.Context) error { return nil }

func (factory) New(context.Context) (glslc.Compiler, error) { return compiler{}, nil }

type compiler struct{}

func (compiler) Compile(ctx context.Context, src []byte) ([]byte, error) {
	return reactor.Compile(ctx, src)
}

func init() {
	driver.Register[glslc.Compiler](factory{})
}
