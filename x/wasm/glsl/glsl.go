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

// Compile turns Vulkan GLSL compute source into SPIR-V.
// src must be GLSL, not SPIR-V.
func Compile(ctx context.Context, src []byte) ([]byte, error) {
	if len(src) == 0 {
		return nil, ErrEmpty
	}
	if IsSPIRV(src) {
		if len(src)%4 != 0 {
			return nil, ErrCompile
		}
		return append([]byte(nil), src...), nil
	}
	return compile(ctx, src)
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
