// Package native compiles compute GLSL with a glslc binary on PATH.
package native

import (
	"context"

	"github.com/lewtec/lewkit/x/driver"
	execdriver "github.com/lewtec/lewkit/x/driver/exec"
	"github.com/lewtec/lewkit/x/driver/glslc"
)

type factory struct{}

func (factory) ID() string   { return "glslc_native" }
func (factory) Name() string { return "native" }
func (factory) Weight() int  { return 80 }

func (factory) CheckCompatibility(ctx context.Context) error {
	return execdriver.RequireBinary(ctx, "glslc")
}

func (factory) New(context.Context) (glslc.Compiler, error) { return compiler{}, nil }

type compiler struct{}

func init() {
	driver.Register[glslc.Compiler](factory{})
}
