package vulkan

import (
	"context"
	"errors"

	drvvulkan "github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/lewtec/lewkit/x/ndarray"
)

const presentGLSL = `#version 450
layout(local_size_x = 16, local_size_y = 16) in;
layout(set = 0, binding = 0) buffer Pix { uint o[]; };
layout(set = 0, binding = 1, rgba8) uniform writeonly image2D dst;
layout(push_constant) uniform Push { int width; int height; int swapRB; int turn; } p;
void main() {
    ivec2 c = ivec2(gl_GlobalInvocationID.xy);
    int dw = p.width;
    int dh = p.height;
    ivec2 s = c;
    if (p.turn == 2 || p.turn == 8) {
        dw = p.height;
        dh = p.width;
    }
    if (p.turn == 2) {
        s = ivec2(c.y, p.height - 1 - c.x);
    } else if (p.turn == 8) {
        s = ivec2(p.width - 1 - c.y, c.x);
    } else if (p.turn == 4) {
        s = ivec2(p.width - 1 - c.x, p.height - 1 - c.y);
    }
    if (c.x >= dw || c.y >= dh || s.x < 0 || s.y < 0 || s.x >= p.width || s.y >= p.height) return;
    int i = (s.y * p.width + s.x) * 4;
    vec4 v = vec4(float(o[i]), float(o[i+1]), float(o[i+2]), float(o[i+3])) * (1.0 / 255.0);
    if (p.swapRB != 0) v = v.bgra;
    imageStore(dst, c, v);
}
`

func (g *gpuEvaluator) presentCode(ctx context.Context) ([]byte, error) {
	g.mu.Lock()
	if g.presentDone {
		code, err := g.present, g.presentErr
		g.mu.Unlock()
		return code, err
	}
	g.mu.Unlock()
	code, err := glsl.Load(ctx, []byte(presentGLSL))
	g.mu.Lock()
	defer g.mu.Unlock()
	if !g.presentDone {
		g.present, g.presentErr, g.presentDone = code, err, true
	}
	return g.present, g.presentErr
}

// Bind returns an evaluator for a caller-owned device. Close does not close the device.
func Bind(device drvvulkan.Device) ndarray.Evaluator {
	return &gpuEvaluator{device: device, own: false}
}

// Paint runs the tensor on the screen's device and presents it. There is no host readback.
func Paint(ctx context.Context, evaluator ndarray.Evaluator, tensor *ndarray.Tensor[uint8], screen drvvulkan.Screen) error {
	gpu, ok := evaluator.(*gpuEvaluator)
	if !ok || gpu == nil || screen == nil || !drvvulkan.Same(gpu.device, screen.Device()) {
		return ndarray.ErrOp
	}
	if tensor == nil {
		return ndarray.ErrOp
	}
	if err := tensor.Resize(tensor.Shape()); err != nil {
		return err
	}
	kernel := tensor.Kernel()
	if kernel == nil || kernel.DType() != ndarray.U8 || len(kernel.Shape()) != 3 || kernel.Shape()[2] != 4 {
		return ndarray.ErrShape
	}
	session, err := gpu.session(ctx, kernel)
	if err != nil {
		return err
	}
	code, err := gpu.presentCode(ctx)
	if err != nil {
		return err
	}
	return session.paint(screen, code)
}

func (s *session) paint(screen drvvulkan.Screen, spirv []byte) error {
	if s == nil || s.kernel == nil || s.device == nil {
		return ndarray.ErrOp
	}
	if s.eval != nil {
		s.eval.Lock()
		defer s.eval.Unlock()
	}
	shape := s.kernel.Shape()
	if err := screen.Fit(shape[1], shape[0]); err != nil {
		return err
	}
	if err := s.prepare(); err != nil {
		return err
	}
	cmd, err := s.device.Begin()
	if err != nil {
		return err
	}
	if err := s.record(cmd); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := cmd.Barrier(); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	if err := screen.JoinPresent(s.output, shape[1], shape[0], spirv); err != nil {
		return errors.Join(err, cmd.Abort())
	}
	return nil
}
