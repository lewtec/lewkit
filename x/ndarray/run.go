package ndarray

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	"github.com/lewtec/lewkit/x/ffi/vulkan"
	"github.com/lewtec/lewkit/x/wasm/glsl"
)

// SPIRV compiles the fused GLSL to SPIR-V.
func (k *Kernel) SPIRV(ctx context.Context) ([]byte, error) {
	if k == nil {
		return nil, ErrOp
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if k.spirv != nil {
		return k.spirv, nil
	}
	spv, err := glsl.Load(ctx, []byte(k.glsl))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOp, err)
	}
	k.spirv = spv
	return spv, nil
}

// Run dispatches the kernel once. dst is binding 0; srcs follow Slots().
func (k *Kernel) Run(ctx context.Context, d *vulkan.Device, dst *vulkan.Buffer, srcs ...*vulkan.Buffer) error {
	if k == nil || d == nil {
		return ErrOp
	}
	if len(srcs) != len(k.slots) {
		return fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.slots), len(srcs))
	}
	if k.n == 0 {
		return nil
	}
	sh, err := k.shader(ctx, d)
	if err != nil {
		return err
	}
	bufs := make([]*vulkan.Buffer, 1+len(srcs))
	bufs[0] = dst
	copy(bufs[1:], srcs)
	var push [4]byte
	binary.LittleEndian.PutUint32(push[:], uint32(k.n))
	groups := uint32((k.n + localSize - 1) / localSize)
	cmd, err := d.Begin()
	if err != nil {
		return err
	}
	if err := cmd.Bind(sh, bufs...); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Push(push[:]); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Dispatch(groups, 1, 1); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Submit(); err != nil {
		return err
	}
	return cmd.Wait()
}

func (k *Kernel) shader(ctx context.Context, d *vulkan.Device) (*vulkan.Shader, error) {
	if k.sh != nil && k.shDev == d {
		return k.sh, nil
	}
	if k.sh != nil {
		if err := k.sh.Close(); err != nil {
			k.sh = nil
			k.shDev = nil
			return nil, err
		}
		k.sh = nil
		k.shDev = nil
	}
	spv, err := k.SPIRV(ctx)
	if err != nil {
		return nil, err
	}
	sh, err := d.Compile(ctx, vulkan.ShaderConfig{
		SPIRV:     spv,
		Bindings:  k.Bindings(),
		PushBytes: 4,
	})
	if err != nil {
		return nil, err
	}
	k.sh = sh
	k.shDev = d
	return sh, nil
}

// Close releases a cached Vulkan shader.
func (k *Kernel) Close() error {
	if k == nil || k.sh == nil {
		return nil
	}
	err := k.sh.Close()
	k.sh = nil
	k.shDev = nil
	return err
}

// Exec allocates host buffers, runs once, and returns the dense output.
func (k *Kernel) Exec(ctx context.Context, d *vulkan.Device, srcs ...[]float32) ([]float32, error) {
	if k == nil {
		return nil, ErrOp
	}
	if len(srcs) != len(k.slots) {
		return nil, fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.slots), len(srcs))
	}
	if k.n == 0 {
		return nil, nil
	}
	dst, err := d.Buffer(max(k.n, 1) * 4)
	if err != nil {
		return nil, err
	}
	defer dst.Close()
	ins := make([]*vulkan.Buffer, len(srcs))
	for i, src := range srcs {
		n := max(len(src), 1)
		b, err := d.Buffer(n * 4)
		if err != nil {
			for _, x := range ins {
				if x != nil {
					x.Close()
				}
			}
			return nil, err
		}
		if err := b.Write(f32bytes(src)); err != nil {
			b.Close()
			for _, x := range ins {
				if x != nil {
					x.Close()
				}
			}
			return nil, err
		}
		ins[i] = b
	}
	defer func() {
		for _, b := range ins {
			b.Close()
		}
	}()
	if err := k.Run(ctx, d, dst, ins...); err != nil {
		return nil, err
	}
	raw := make([]byte, k.n*4)
	if err := dst.Read(raw); err != nil {
		return nil, err
	}
	if k.outDT == I32 {
		out := make([]float32, k.n)
		for i := range out {
			out[i] = float32(int32(binary.LittleEndian.Uint32(raw[i*4:])))
		}
		return out, nil
	}
	return bytesF32(raw), nil
}

func f32bytes(v []float32) []byte {
	b := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

func bytesF32(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}
