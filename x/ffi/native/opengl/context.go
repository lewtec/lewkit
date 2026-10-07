//go:build !darwin && !ios

package opengl

import (
	"fmt"
	"runtime"
)

// glContext is one current-thread OpenGL context.
// Make and Unmake run on the thread that owns it. Proc resolves a GL entry
// point while the context is current. Destroy releases the native context.
type glContext interface {
	Make() error
	Unmake()
	Swap() error
	Destroy()
	Proc(name string) uintptr
	GLES() bool
}

// retargeter moves a window context onto a replacement native handle.
// The call runs on the context thread and leaves the context current.
type retargeter interface {
	Retarget(native uintptr, width, height int) error
}

// runner keeps one context current on one OS thread.
// A GL context cannot follow a goroutine across threads.
type runner struct {
	ctx  glContext
	jobs chan func()
	done chan struct{}
}

func startRunner(ctx glContext) (*runner, error) {
	if ctx == nil {
		return nil, ErrUnavailable
	}
	r := &runner{ctx: ctx, jobs: make(chan func()), done: make(chan struct{})}
	ready := make(chan error, 1)
	go func() {
		runtime.LockOSThread()
		if err := ctx.Make(); err != nil {
			ctx.Destroy()
			ready <- err
			return
		}
		ready <- nil
		for fn := range r.jobs {
			fn()
		}
		ctx.Unmake()
		ctx.Destroy()
		close(r.done)
	}()
	if err := <-ready; err != nil {
		return nil, err
	}
	return r, nil
}

// Do runs fn on the context thread.
func (r *runner) Do(fn func() error) (err error) {
	if r == nil {
		return ErrClosed
	}
	done := make(chan struct{})
	r.jobs <- func() {
		defer close(done)
		defer func() {
			if rec := recover(); rec != nil && err == nil {
				err = fmt.Errorf("%w: %v", ErrUnavailable, rec)
			}
		}()
		err = fn()
	}
	<-done
	return err
}

// Stop drains the thread and destroys the context.
func (r *runner) Stop() {
	if r == nil {
		return
	}
	close(r.jobs)
	<-r.done
}
