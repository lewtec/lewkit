// Package run is the evaluation sequence the GPU sessions share.
// A backend still compiles its kernel, grows its buffers, and writes its own grow log.
package run

import (
	"context"
	"errors"
	"sync"

	"github.com/lewtec/lewkit/x/driver/ndeval/internal/slots"
	"github.com/lewtec/lewkit/x/ndarray"
)

// Eval checks the context when the backend does, then the kernel and the device,
// then the pipeline under the eval lock. A zero-length kernel returns before dispatch.
// Vulkan passes checkCtx false so a cancelled context still reaches the device check.
func Eval[B slots.Buffer](
	ctx context.Context,
	checkCtx bool,
	kernel *ndarray.Kernel,
	deviceOK bool,
	eval *sync.Mutex,
	ready bool,
	set *slots.Set[B],
	output []byte,
	dispatch func() error,
) error {
	if checkCtx {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if kernel == nil || !deviceOK {
		return ndarray.ErrOp
	}
	if eval != nil {
		eval.Lock()
		defer eval.Unlock()
	}
	if !ready {
		return ndarray.ErrOp
	}
	if err := slots.CheckOutput(kernel, output); err != nil {
		return err
	}
	if kernel.Size() == 0 {
		return nil
	}
	if err := dispatch(); err != nil {
		return err
	}
	return set.ReadHost(kernel, output)
}

// Prepare fits the buffers, then stores each input.
func Prepare[B slots.Buffer](fit func() error, set *slots.Set[B], kernel *ndarray.Kernel) error {
	if err := fit(); err != nil {
		return err
	}
	return set.Load(kernel)
}

// Groups prepares, fills the push constants, and skips a zero group count.
// N is int for Metal and Direct3D 12, and uint32 for OpenGL.
func Groups[B slots.Buffer, N interface{ ~int | ~uint32 }](
	prepare func() error,
	set *slots.Set[B],
	kernel *ndarray.Kernel,
	run func([]B, []byte, N) error,
) error {
	if err := prepare(); err != nil {
		return err
	}
	push := set.FillPush(kernel)
	groups := N(kernel.Groups())
	if groups < 1 {
		return nil
	}
	return run(set.Bound, push, groups)
}

// Close runs forget once, closes the pipeline, and joins the buffer close.
// The caller returns before this when the session is nil.
func Close[B slots.Buffer](eval *sync.Mutex, forget *func(), closePipe func() error, set *slots.Set[B]) error {
	if eval != nil {
		eval.Lock()
		defer eval.Unlock()
	}
	if forget != nil && *forget != nil {
		(*forget)()
		*forget = nil
	}
	var err error
	if closePipe != nil {
		err = closePipe()
	}
	return errors.Join(err, set.Close())
}
