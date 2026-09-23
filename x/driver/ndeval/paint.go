package ndeval

import (
	"context"
	"sync"

	"github.com/lewtec/lewkit/x/driver/vulkan"
	"github.com/lewtec/lewkit/x/ffi/wasm/glsl"
	"github.com/lewtec/lewkit/x/ndarray"
)

const presentGLSL = `#version 450
layout(local_size_x = 16, local_size_y = 16) in;
layout(set = 0, binding = 0) buffer Pix { uint o[]; };
layout(set = 0, binding = 1, rgba8) uniform writeonly image2D dst;
layout(push_constant) uniform Push { int width; int height; int swapRB; } p;
void main() {
    ivec2 c = ivec2(gl_GlobalInvocationID.xy);
    if (c.x >= p.width || c.y >= p.height) return;
    int i = (c.y * p.width + c.x) * 4;
    vec4 v = vec4(float(o[i]), float(o[i+1]), float(o[i+2]), float(o[i+3])) * (1.0 / 255.0);
    if (p.swapRB != 0) v = v.bgra;
    imageStore(dst, c, v);
}
`

var presentSPIRV struct {
	sync.Mutex
	code []byte
	err  error
	done bool
}

func presentCode(ctx context.Context) ([]byte, error) {
	presentSPIRV.Lock()
	defer presentSPIRV.Unlock()
	if presentSPIRV.done {
		return presentSPIRV.code, presentSPIRV.err
	}
	presentSPIRV.done = true
	presentSPIRV.code, presentSPIRV.err = glsl.Load(ctx, []byte(presentGLSL))
	return presentSPIRV.code, presentSPIRV.err
}

// Bind returns an evaluator for a caller-owned device. Close does not close the device.
func Bind(device vulkan.Device) ndarray.Evaluator {
	return &gpuEvaluator{device: device, own: false}
}

// Paint runs the tensor on the screen's device and presents it. There is no host readback.
func Paint(ctx context.Context, evaluator ndarray.Evaluator, tensor *ndarray.Tensor[uint8], screen vulkan.Screen) error {
	gpu, ok := evaluator.(*gpuEvaluator)
	if !ok || gpu == nil || screen == nil || !vulkan.Same(gpu.device, screen.Device()) {
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
	code, err := presentCode(ctx)
	if err != nil {
		return err
	}
	return session.paint(screen, code)
}

func (s *session) paint(screen vulkan.Screen, spirv []byte) error {
	if s == nil || s.kernel == nil || s.device == nil {
		return ndarray.ErrOp
	}
	if s.eval != nil {
		s.eval.Lock()
		defer s.eval.Unlock()
	}
	if err := s.dispatch(); err != nil {
		return err
	}
	shape := s.kernel.Shape()
	return screen.Present(s.output, shape[1], shape[0], spirv)
}
