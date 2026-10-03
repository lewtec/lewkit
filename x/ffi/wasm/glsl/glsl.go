package glsl

import (
	"context"
	"encoding/binary"
	"errors"
)

var (
	// ErrCompile means GLSL did not become SPIR-V.
	ErrCompile = errors.New("glsl compile")
	// ErrEmpty means the source is empty.
	ErrEmpty = errors.New("empty shader")
)

// IsSPIRV reports whether b starts with the SPIR-V magic number.
func IsSPIRV(b []byte) bool {
	return len(b) >= 4 && binary.LittleEndian.Uint32(b[:4]) == 0x07230203
}

// Stage is a shader stage the embedded compiler can emit.
type Stage int

const (
	// StageVertex is a vertex shader.
	StageVertex Stage = 0
	// StageFragment is a fragment shader.
	StageFragment Stage = 4
	// StageCompute is a compute shader.
	StageCompute Stage = 5
)

// Compile turns Vulkan GLSL compute source into SPIR-V.
// src must be GLSL, not SPIR-V.
// A registered hash returns that SPIR-V and does not run a compiler.
func Compile(ctx context.Context, src []byte) ([]byte, error) {
	return CompileStage(ctx, StageCompute, src)
}

// CompileStage turns Vulkan GLSL for stage into SPIR-V.
// A registered hash returns that SPIR-V and does not run a compiler.
// A missing hash runs glslc when it is on PATH, then the embedded glslang.
func CompileStage(ctx context.Context, stage Stage, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmpty
	}
	if !knownStage(stage) {
		return nil, ErrCompile
	}
	if IsSPIRV(src) {
		if len(src)%4 != 0 {
			return nil, ErrCompile
		}
		return append([]byte(nil), src...), nil
	}
	if spv, ok := Lookup(stage, src); ok {
		return spv, nil
	}
	return compileStage(ctx, stage, src)
}

// CompileGlslang compiles src with the embedded glslang reactor.
// It does not read the registry and it does not run glslc.
// Codegen uses it so the committed SPIR-V does not depend on the host.
func CompileGlslang(ctx context.Context, stage Stage, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmpty
	}
	if !knownStage(stage) {
		return nil, ErrCompile
	}
	if IsSPIRV(src) {
		if len(src)%4 != 0 {
			return nil, ErrCompile
		}
		return append([]byte(nil), src...), nil
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return compileSPIRV(ctx, stage, src)
}

// Load returns SPIR-V. SPIR-V is copied; GLSL is compiled.
func Load(ctx context.Context, src []byte) ([]byte, error) {
	if IsSPIRV(src) {
		if len(src)%4 != 0 {
			return nil, ErrCompile
		}
		return append([]byte(nil), src...), nil
	}
	return Compile(ctx, src)
}
