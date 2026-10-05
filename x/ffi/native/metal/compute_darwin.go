//go:build darwin

package metal

import (
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego/objc"
	"github.com/lewtec/lewkit/x/ffi/native"
)

// MTLCommandBufferStatusCompleted and MTLCommandBufferStatusError.
// storageShared is MTLResourceStorageModeShared. The CPU reads contents
// after the command buffer completes. Managed storage is not used.
const (
	statusCompleted = 4
	statusError     = 5
	storageShared   = 0
)

// Device is the system Metal device used for compute.
// Close releases this queue. The process-wide device stays for Screen.
type Device struct {
	mu     sync.Mutex
	id     objc.ID
	queue  objc.ID
	name   string
	closed bool
}

// Pipeline is one compute kernel compiled from Metal shading language.
type Pipeline struct {
	d       *Device
	lib     objc.ID
	fn      objc.ID
	pipe    objc.ID
	threads int
}

// Buffer is a shared storage buffer. contents is readable after Dispatch waits.
type Buffer struct {
	d    *Device
	id   objc.ID
	size int
}

type mtlSize struct {
	Width, Height, Depth uintptr
}

// OpenDevice opens the system default device and a command queue.
func OpenDevice() (*Device, error) {
	raw, err := systemDevice()
	if err != nil {
		return nil, err
	}
	d := &Device{id: raw.Send(selRetain)}
	var openErr error
	withPool(func() {
		queue := d.id.Send(selNewCommandQueue)
		if queue == 0 {
			openErr = ErrUnavailable
			return
		}
		d.queue = queue
		name := d.id.Send(selName)
		if name != 0 {
			d.name = native.GoString(uintptr(name.Send(selUTF8)))
		}
	})
	if openErr != nil {
		_ = d.Close()
		return nil, openErr
	}
	return d, nil
}

// Name is the device name, or empty.
func (d *Device) Name() string {
	if d == nil {
		return ""
	}
	return d.name
}

// Close releases the queue and this device reference.
func (d *Device) Close() error {
	if d == nil {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed {
		return nil
	}
	d.closed = true
	withPool(func() {
		if d.queue != 0 {
			d.queue.Send(selRelease)
		}
		if d.id != 0 {
			d.id.Send(selRelease)
		}
	})
	d.queue = 0
	d.id = 0
	return nil
}

// Buffer allocates a shared buffer of size bytes.
func (d *Device) Buffer(size int) (*Buffer, error) {
	if d == nil {
		return nil, ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.id == 0 {
		return nil, ErrClosed
	}
	if size < 1 {
		return nil, ErrSize
	}
	buf := d.id.Send(selNewBufferLen, uintptr(size), uintptr(storageShared))
	if buf == 0 {
		return nil, ErrUnavailable
	}
	return &Buffer{d: d, id: buf, size: size}, nil
}

// Len is the requested size in bytes.
func (b *Buffer) Len() int {
	if b == nil {
		return 0
	}
	return b.size
}

// Store copies p into the start of the buffer.
func (b *Buffer) Store(p []byte) error {
	return b.copy(p, true)
}

// Read copies the buffer into p.
func (b *Buffer) Read(p []byte) error {
	return b.copy(p, false)
}

func (b *Buffer) copy(p []byte, into bool) error {
	if b == nil || b.d == nil {
		return ErrClosed
	}
	b.d.mu.Lock()
	defer b.d.mu.Unlock()
	if b.id == 0 || b.d.closed {
		return ErrClosed
	}
	if len(p) > b.size {
		return ErrSize
	}
	ptr := uintptr(b.id.Send(selContents))
	if ptr == 0 {
		return ErrUnavailable
	}
	raw := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), b.size)
	if into {
		copy(raw, p)
	} else {
		copy(p, raw)
	}
	return nil
}

// Close releases the buffer.
func (b *Buffer) Close() error {
	if b == nil || b.d == nil {
		return nil
	}
	b.d.mu.Lock()
	defer b.d.mu.Unlock()
	if b.id == 0 {
		return nil
	}
	id := b.id
	b.id = 0
	id.Send(selRelease)
	return nil
}

// Compile builds a compute pipeline. threads is the threadgroup width.
// source is Metal shading language. The kernel function is ndeval.
func (d *Device) Compile(source string, threads int) (*Pipeline, error) {
	if d == nil {
		return nil, ErrClosed
	}
	if threads < 1 || threads > 1024 || source == "" {
		return nil, ErrSize
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.id == 0 {
		return nil, ErrClosed
	}
	var pipe *Pipeline
	var err error
	withPool(func() {
		pipe, err = d.compile(source, threads)
	})
	return pipe, err
}

func (d *Device) compile(source string, threads int) (*Pipeline, error) {
	if newLib == nil {
		native.Register(&newLib, msgSend())
	}
	if newCompute == nil {
		native.Register(&newCompute, msgSend())
	}
	if newLib == nil || newCompute == nil {
		return nil, ErrUnavailable
	}
	var nsErr objc.ID
	lib := newLib(d.id, selNewLibrary, nsstr(source), 0, &nsErr)
	if lib == 0 {
		return nil, compileErr(nsErr)
	}
	fn := lib.Send(selNewFunction, nsstr("ndeval"))
	if fn == 0 {
		lib.Send(selRelease)
		return nil, fmt.Errorf("%w: ndeval", ErrUnavailable)
	}
	nsErr = 0
	state := newCompute(d.id, selNewCompute, fn, &nsErr)
	if state == 0 {
		fn.Send(selRelease)
		lib.Send(selRelease)
		return nil, compileErr(nsErr)
	}
	return &Pipeline{d: d, lib: lib, fn: fn, pipe: state, threads: threads}, nil
}

func compileErr(nsErr objc.ID) error {
	detail := ""
	if nsErr != 0 {
		detail = native.GoString(uintptr(nsErr.Send(selLocalized).Send(selUTF8)))
	}
	if detail == "" {
		return fmt.Errorf("%w: library", ErrUnavailable)
	}
	return fmt.Errorf("%w: %s", ErrUnavailable, detail)
}

// Dispatch runs the kernel over groups threadgroups and waits until it finishes.
// push is copied before the call returns. buffers[i] is bound at index i.
// push is bound at the next index.
func (p *Pipeline) Dispatch(buffers []*Buffer, push []byte, groups int) error {
	if p == nil || p.d == nil {
		return ErrClosed
	}
	if groups < 1 || len(push) == 0 || len(push)%4 != 0 {
		return ErrSize
	}
	p.d.mu.Lock()
	defer p.d.mu.Unlock()
	if p.pipe == 0 || p.d.closed || p.d.queue == 0 {
		return ErrClosed
	}
	for _, b := range buffers {
		if b == nil || b.id == 0 || b.d != p.d {
			return ErrClosed
		}
	}
	if dispatchFn == nil {
		native.Register(&dispatchFn, msgSend())
	}
	if dispatchFn == nil {
		return ErrUnavailable
	}
	var runErr error
	withPool(func() {
		cmd := p.d.queue.Send(selCommandBuffer)
		if cmd == 0 {
			runErr = ErrUnavailable
			return
		}
		enc := cmd.Send(selComputeEncoder)
		if enc == 0 {
			runErr = ErrUnavailable
			return
		}
		enc.Send(selSetCompute, p.pipe)
		for i, b := range buffers {
			enc.Send(selSetComputeBuffer, b.id, uintptr(0), uintptr(i))
		}
		enc.Send(selSetComputeBytes, unsafe.Pointer(&push[0]), uintptr(len(push)), uintptr(len(buffers)))
		dispatchFn(enc, selDispatch,
			mtlSize{Width: uintptr(groups), Height: 1, Depth: 1},
			mtlSize{Width: uintptr(p.threads), Height: 1, Depth: 1},
		)
		enc.Send(selEndEncoding)
		cmd.Send(selCommit)
		cmd.Send(selWait)
		status := int(cmd.Send(selStatus))
		if status == statusError {
			runErr = compileErr(cmd.Send(selCmdError))
			return
		}
		if status != statusCompleted {
			runErr = fmt.Errorf("%w: command status %d", ErrUnavailable, status)
		}
	})
	return runErr
}

// Close releases the pipeline, function, and library.
func (p *Pipeline) Close() error {
	if p == nil || p.d == nil {
		return nil
	}
	p.d.mu.Lock()
	defer p.d.mu.Unlock()
	withPool(func() {
		for _, id := range []objc.ID{p.pipe, p.fn, p.lib} {
			if id != 0 {
				id.Send(selRelease)
			}
		}
	})
	p.pipe, p.fn, p.lib = 0, 0, 0
	return nil
}

var (
	newCompute func(objc.ID, objc.SEL, objc.ID, *objc.ID) objc.ID
	dispatchFn func(objc.ID, objc.SEL, mtlSize, mtlSize)
)

var (
	selName             = objc.RegisterName("name")
	selNewBufferLen     = objc.RegisterName("newBufferWithLength:options:")
	selNewCompute       = objc.RegisterName("newComputePipelineStateWithFunction:error:")
	selComputeEncoder   = objc.RegisterName("computeCommandEncoder")
	selSetCompute       = objc.RegisterName("setComputePipelineState:")
	selSetComputeBuffer = objc.RegisterName("setBuffer:offset:atIndex:")
	selSetComputeBytes  = objc.RegisterName("setBytes:length:atIndex:")
	selDispatch         = objc.RegisterName("dispatchThreadgroups:threadsPerThreadgroup:")
	selStatus           = objc.RegisterName("status")
	selCmdError         = objc.RegisterName("error")
)
