//go:build !darwin && !ios

package opengl

import (
	"encoding/binary"
	"fmt"
	"log/slog"
	"sync"
	"unsafe"
)

// Device runs one compute kernel. It does not attach a surface.
type Device struct {
	mu     sync.Mutex
	run    *runner
	gl     *glAPI
	name   string
	limit  [3]uint32
	closed bool
}

// Buffer is one shader-storage block.
type Buffer struct {
	dev *Device
	id  uint32
	n   int
}

// Program is one compiled compute kernel.
type Program struct {
	dev       *Device
	id        uint32
	n, d0, d1 int32
	d2, d3    int32
}

func openCompute(ctx glContext) (*Device, error) {
	run, err := startRunner(ctx)
	if err != nil {
		return nil, err
	}
	d := &Device{run: run}
	if err := run.Do(func() error {
		api, err := loadAPI(ctx)
		if err != nil {
			return err
		}
		d.gl = api
		d.name = api.renderer
		d.limit = api.workGroupLimit()
		if !api.compute {
			return ErrUnavailable
		}
		slog.Debug("opengl device", "renderer", api.renderer, "major", api.major, "minor", api.minor, "gles", api.gles)
		return nil
	}); err != nil {
		d.Close()
		return nil, err
	}
	return d, nil
}

// Name is the GL renderer string.
func (d *Device) Name() string {
	if d == nil || d.name == "" {
		return "opengl"
	}
	return d.name
}

// GLES reports whether the device is OpenGL ES.
func (d *Device) GLES() bool {
	return d != nil && d.gl != nil && d.gl.gles
}

// Buffer allocates n bytes of shader storage. n is at least 4.
func (d *Device) Buffer(n int) (*Buffer, error) {
	if d == nil {
		return nil, ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.run == nil {
		return nil, ErrClosed
	}
	if n < 4 {
		n = 4
	}
	var buf Buffer
	err := d.run.Do(func() error {
		var id uint32
		d.gl.genBuffers(1, &id)
		d.gl.bindBuffer(glShaderStorageBuffer, id)
		d.gl.bufferData(glShaderStorageBuffer, n, 0, glDynamicDraw)
		if err := d.gl.check("buffer"); err != nil {
			d.gl.deleteBuffers(1, &id)
			return err
		}
		buf = Buffer{dev: d, id: id, n: n}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &buf, nil
}

// Len is the allocated size.
func (b *Buffer) Len() int {
	if b == nil {
		return 0
	}
	return b.n
}

// Store copies p into the buffer. p longer than the buffer is truncated.
func (b *Buffer) Store(p []byte) error {
	if b == nil || b.dev == nil {
		return ErrClosed
	}
	b.dev.mu.Lock()
	defer b.dev.mu.Unlock()
	if b.dev.closed || b.id == 0 {
		return ErrClosed
	}
	if len(p) > b.n {
		p = p[:b.n]
	}
	return b.dev.run.Do(func() error {
		g := b.dev.gl
		g.bindBuffer(glShaderStorageBuffer, b.id)
		ptr := g.mapBuffer(glShaderStorageBuffer, 0, b.n, glMapWriteBit|glMapInvalidateBufferBit)
		if ptr == 0 {
			return g.check("map write")
		}
		dst := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), b.n)
		clear(dst)
		copy(dst, p)
		if g.unmapBuffer(glShaderStorageBuffer) == 0 {
			return ErrUnavailable
		}
		return nil
	})
}

// Read copies the buffer into p.
func (b *Buffer) Read(p []byte) error {
	if b == nil || b.dev == nil {
		return ErrClosed
	}
	b.dev.mu.Lock()
	defer b.dev.mu.Unlock()
	if b.dev.closed || b.id == 0 {
		return ErrClosed
	}
	if len(p) > b.n {
		p = p[:b.n]
	}
	return b.dev.run.Do(func() error {
		g := b.dev.gl
		g.barrier(glBufferUpdateBarrierBit | glShaderStorageBarrierBit)
		g.bindBuffer(glShaderStorageBuffer, b.id)
		ptr := g.mapBuffer(glShaderStorageBuffer, 0, b.n, glMapReadBit)
		if ptr == 0 {
			return g.check("map read")
		}
		src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), len(p))
		copy(p, src)
		if g.unmapBuffer(glShaderStorageBuffer) == 0 {
			return ErrUnavailable
		}
		return nil
	})
}

// Close deletes the buffer.
func (b *Buffer) Close() error {
	if b == nil || b.dev == nil || b.id == 0 {
		return nil
	}
	b.dev.mu.Lock()
	defer b.dev.mu.Unlock()
	if b.dev.closed || b.dev.run == nil || b.id == 0 {
		return nil
	}
	id := b.id
	b.id = 0
	return b.dev.run.Do(func() error {
		b.dev.gl.deleteBuffers(1, &id)
		return nil
	})
}

// Compile builds one compute shader.
func (d *Device) Compile(src string) (*Program, error) {
	if d == nil {
		return nil, ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.run == nil {
		return nil, ErrClosed
	}
	var prog Program
	err := d.run.Do(func() error {
		sh, err := d.gl.compile(glComputeShader, src)
		if err != nil {
			return err
		}
		id := d.gl.createProgram()
		if id == 0 {
			d.gl.deleteShader(sh)
			return ErrUnavailable
		}
		d.gl.attachShader(id, sh)
		d.gl.linkProgram(id)
		var ok int32
		d.gl.getProgramiv(id, glLinkStatus, &ok)
		d.gl.deleteShader(sh)
		if ok == 0 {
			detail := d.gl.log(false, id)
			d.gl.deleteProgram(id)
			if detail == "" {
				detail = "compute link"
			}
			return fmt.Errorf("%w: %s", ErrUnavailable, detail)
		}
		prog = Program{
			dev: d,
			id:  id,
			n:   d.gl.uniform(id, "n"),
			d0:  d.gl.uniform(id, "d0"),
			d1:  d.gl.uniform(id, "d1"),
			d2:  d.gl.uniform(id, "d2"),
			d3:  d.gl.uniform(id, "d3"),
		}
		return d.gl.check("compile")
	})
	if err != nil {
		return nil, err
	}
	return &prog, nil
}

// Run dispatches groups of the kernel over buffers. push is n, d0, d1, d2, d3
// as five little-endian uint32 values.
func (d *Device) Run(prog *Program, groups uint32, buffers []*Buffer, push []byte) error {
	if d == nil || prog == nil || prog.id == 0 {
		return ErrClosed
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.closed || d.run == nil {
		return ErrClosed
	}
	return d.run.Do(func() error {
		g := d.gl
		g.useProgram(prog.id)
		set := func(loc int32, word uint32) {
			if loc >= 0 && g.uniform1ui != nil {
				g.uniform1ui(loc, word)
			}
		}
		words := pushWords(push)
		set(prog.n, words[0])
		set(prog.d0, words[1])
		set(prog.d1, words[2])
		set(prog.d2, words[3])
		set(prog.d3, words[4])
		for i, buf := range buffers {
			if buf == nil || buf.id == 0 {
				continue
			}
			g.bindBufferBase(glShaderStorageBuffer, uint32(i), buf.id)
		}
		if err := g.check("bind"); err != nil {
			return err
		}
		if groups == 0 {
			groups = 1
		}
		x, y, z, err := coverGroups(groups, d.limit)
		if err != nil {
			return err
		}
		g.dispatch(x, y, z)
		g.barrier(glShaderStorageBarrierBit | glBufferUpdateBarrierBit)
		if err := g.check("dispatch"); err != nil {
			return fmt.Errorf("%w (%d groups as %d,%d,%d)", err, groups, x, y, z)
		}
		return nil
	})
}

// coverGroups spreads a 1D group count across X, then Y, then Z.
// Each axis stays within the device maximum. The shader lines the
// invocation id back up, so element i is still invocation i.
func coverGroups(groups uint32, max [3]uint32) (uint32, uint32, uint32, error) {
	if groups == 0 {
		return 0, 0, 0, fmt.Errorf("%w: no groups", ErrUnavailable)
	}
	for i := range max {
		if max[i] < 1 {
			max[i] = 65535
		}
	}
	if groups <= max[0] {
		return groups, 1, 1, nil
	}
	x := max[0]
	y64 := (uint64(groups) + uint64(x) - 1) / uint64(x)
	if y64 <= uint64(max[1]) {
		return x, uint32(y64), 1, nil
	}
	y := max[1]
	plane := uint64(x) * uint64(y)
	z64 := (uint64(groups) + plane - 1) / plane
	if z64 > uint64(max[2]) {
		return 0, 0, 0, fmt.Errorf("%w: dispatch %d groups", ErrUnavailable, groups)
	}
	return x, y, uint32(z64), nil
}

func pushWords(push []byte) [5]uint32 {
	var out [5]uint32
	for i := range out {
		off := i * 4
		if off+4 > len(push) {
			break
		}
		out[i] = binary.LittleEndian.Uint32(push[off:])
	}
	return out
}

// Close releases the device.
func (p *Program) Close() error {
	if p == nil || p.dev == nil || p.id == 0 {
		return nil
	}
	p.dev.mu.Lock()
	defer p.dev.mu.Unlock()
	if p.dev.closed || p.dev.run == nil || p.id == 0 {
		return nil
	}
	id := p.id
	p.id = 0
	return p.dev.run.Do(func() error {
		p.dev.gl.deleteProgram(id)
		return nil
	})
}

// Close releases the context.
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
	if d.run != nil {
		d.run.Stop()
		d.run = nil
	}
	return nil
}
