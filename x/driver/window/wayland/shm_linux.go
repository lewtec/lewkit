//go:build linux && !android

package wayland

import (
	"fmt"
	"image"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/driver/window"
	"golang.org/x/sys/unix"
)

type shmSlot struct {
	id   uint32
	busy int32
}

func (w *wlwin) put() error {
	if atomic.LoadInt32(&w.closed) != 0 {
		return window.ErrClosed
	}
	w.blit.Lock()
	defer w.blit.Unlock()
	if atomic.LoadInt32(&w.closed) != 0 || w.conn == nil {
		return window.ErrClosed
	}
	var width, height int
	w.WithFront(func(src *image.RGBA) {
		if src == nil {
			return
		}
		width, height = src.Rect.Dx(), src.Rect.Dy()
		if width < 1 || height < 1 || src.Stride != width*4 {
			width, height = 0, 0
			return
		}
		n := width * height * 4
		if cap(w.bgra) < n {
			w.bgra = make([]byte, n)
		}
		w.bgra = w.bgra[:n]
		window.ToBGRA(w.bgra, src)
	})
	if width < 1 || height < 1 || len(w.bgra) < width*height*4 {
		return nil
	}
	if w.poolW != width || w.poolH != height {
		if !w.slotsFree() {
			return nil
		}
		if err := w.realloc(width, height); err != nil {
			return err
		}
	}
	slot := -1
	for i := range w.slots {
		if atomic.CompareAndSwapInt32(&w.slots[i].busy, 0, 1) {
			slot = i
			break
		}
	}
	if slot < 0 {
		return nil
	}
	stride := width * 4
	off := slot * stride * height
	copy(w.mem[off:off+stride*height], w.bgra[:stride*height])
	if err := w.conn.send(w.surface, 1, pack(w.slots[slot].id, int32(0), int32(0))); err != nil {
		atomic.StoreInt32(&w.slots[slot].busy, 0)
		return err
	}
	if err := w.conn.send(w.surface, 2, pack(int32(0), int32(0), int32(width), int32(height))); err != nil {
		return err
	}
	return w.conn.send(w.surface, 6, nil)
}

func (w *wlwin) slotsFree() bool {
	for i := range w.slots {
		if atomic.LoadInt32(&w.slots[i].busy) != 0 {
			return false
		}
	}
	return true
}

func (w *wlwin) realloc(width, height int) error {
	if width > 16384 || height > 16384 || !w.haveFormat {
		return fmt.Errorf("wayland: size %dx%d", width, height)
	}
	stride := width * 4
	size := stride * height * len(w.slots)
	for i := range w.slots {
		if w.slots[i].id != 0 {
			_ = w.conn.send(w.slots[i].id, 0, nil)
			w.conn.remove(w.slots[i].id)
			w.slots[i].id = 0
		}
	}
	if w.pool != 0 {
		_ = w.conn.send(w.pool, 1, nil)
		w.conn.remove(w.pool)
		w.pool = 0
	}
	if w.mem != nil {
		_ = unix.Munmap(w.mem)
		w.mem = nil
	}
	if w.poolFD >= 0 {
		_ = unix.Close(w.poolFD)
		w.poolFD = -1
	}
	fd, err := unix.MemfdCreate("lewkit-wl", unix.MFD_CLOEXEC)
	if err != nil {
		return err
	}
	if err := unix.Ftruncate(fd, int64(size)); err != nil {
		_ = unix.Close(fd)
		return err
	}
	mem, err := unix.Mmap(fd, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		_ = unix.Close(fd)
		return err
	}
	id := w.conn.alloc(nil)
	if err := w.conn.send(w.shm, 0, pack(id, int32(size)), fd); err != nil {
		_ = unix.Munmap(mem)
		_ = unix.Close(fd)
		return err
	}
	w.pool = id
	w.poolFD = fd
	w.mem = mem
	w.poolW = width
	w.poolH = height
	for i := range w.slots {
		buf := w.conn.alloc(w.onBuffer(i))
		off := int32(i * stride * height)
		err := w.conn.send(w.pool, 0, pack(buf, off, int32(width), int32(height), int32(stride), w.format))
		if err != nil {
			return err
		}
		w.slots[i].id = buf
		atomic.StoreInt32(&w.slots[i].busy, 0)
	}
	return nil
}

func (w *wlwin) onBuffer(slot int) func(uint16, []byte) {
	return func(op uint16, _ []byte) {
		if op != 0 {
			return
		}
		atomic.StoreInt32(&w.slots[slot].busy, 0)
	}
}
