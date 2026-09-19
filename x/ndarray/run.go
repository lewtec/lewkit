package ndarray

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"log/slog"
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
	spirv, err := glsl.Load(ctx, []byte(k.glsl))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrOp, err)
	}
	k.spirv = spirv
	slog.Debug("ndarray spirv", "bytes", len(spirv))
	return spirv, nil
}

// Run dispatches the kernel once. output is binding 0; inputs follow Slots().
func (k *Kernel) Run(ctx context.Context, device *vulkan.Device, output *vulkan.Buffer, inputs ...*vulkan.Buffer) error {
	if k == nil || device == nil {
		return ErrOp
	}
	if len(inputs) != len(k.bufs) {
		return fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.bufs), len(inputs))
	}
	if k.size == 0 {
		return nil
	}
	shader, err := k.ensurePipeline(ctx, device)
	if err != nil {
		return err
	}
	need := 1 + len(inputs)
	if cap(k.runBuffers) < need {
		k.runBuffers = make([]*vulkan.Buffer, need)
	} else {
		k.runBuffers = k.runBuffers[:need]
	}
	k.runBuffers[0] = output
	copy(k.runBuffers[1:], inputs)
	bufs := k.runBuffers
	var push [pushBytes]byte
	binary.LittleEndian.PutUint32(push[0:], uint32(k.size))
	for i, s := range k.shape {
		if i >= 4 {
			break
		}
		binary.LittleEndian.PutUint32(push[4+4*i:], uint32(s))
	}
	groups := uint32((k.size + localSize - 1) / localSize)
	cmd, err := device.Begin()
	if err != nil {
		return err
	}
	if err := cmd.Bind(shader, bufs...); err != nil {
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

func (k *Kernel) ensurePipeline(ctx context.Context, device *vulkan.Device) (*vulkan.Shader, error) {
	if k.pipeline != nil && k.pipelineDevice == device {
		return k.pipeline, nil
	}
	if k.pipeline != nil {
		if err := k.pipeline.Close(); err != nil {
			k.pipeline = nil
			k.pipelineDevice = nil
			return nil, err
		}
		k.pipeline = nil
		k.pipelineDevice = nil
	}
	spirv, err := k.SPIRV(ctx)
	if err != nil {
		return nil, err
	}
	shader, err := device.Compile(ctx, vulkan.ShaderConfig{
		SPIRV:     spirv,
		Bindings:  k.Bindings(),
		PushBytes: pushBytes,
	})
	if err != nil {
		return nil, err
	}
	k.pipeline = shader
	k.pipelineDevice = device
	return shader, nil
}

// Close releases a cached Vulkan shader.
func (k *Kernel) Close() error {
	if k == nil || k.pipeline == nil {
		return nil
	}
	err := k.pipeline.Close()
	k.pipeline = nil
	k.pipelineDevice = nil
	return err
}

// Exec allocates host buffers, runs once, and returns the dense output.
func (k *Kernel) Exec(ctx context.Context, device *vulkan.Device, inputs ...[]float32) ([]float32, error) {
	if k == nil {
		return nil, ErrOp
	}
	if len(inputs) != len(k.bufs) {
		return nil, fmt.Errorf("%w: want %d inputs, got %d", ErrOp, len(k.bufs), len(inputs))
	}
	if k.size == 0 {
		return nil, nil
	}
	output, err := device.Buffer(max(k.size, 1) * 4)
	if err != nil {
		return nil, err
	}
	defer output.Close()
	bufs := make([]*vulkan.Buffer, len(inputs))
	for i, src := range inputs {
		n := max(len(src), 1)
		b, err := device.Buffer(n * 4)
		if err != nil {
			for _, x := range bufs {
				if x != nil {
					x.Close()
				}
			}
			return nil, err
		}
		if err := b.Write(floatBytes(src)); err != nil {
			b.Close()
			for _, x := range bufs {
				if x != nil {
					x.Close()
				}
			}
			return nil, err
		}
		bufs[i] = b
	}
	defer func() {
		for _, b := range bufs {
			b.Close()
		}
	}()
	if err := k.Run(ctx, device, output, bufs...); err != nil {
		return nil, err
	}
	raw := make([]byte, k.size*4)
	if err := output.Read(raw); err != nil {
		return nil, err
	}
	if k.outType == I32 {
		out := make([]float32, k.size)
		for i := range out {
			out[i] = float32(int32(binary.LittleEndian.Uint32(raw[i*4:])))
		}
		return out, nil
	}
	return bytesToFloat32(raw), nil
}

func floatBytes(v []float32) []byte {
	b := make([]byte, len(v)*4)
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(x))
	}
	return b
}

func bytesToFloat32(b []byte) []float32 {
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out
}
