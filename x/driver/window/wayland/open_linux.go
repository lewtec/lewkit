//go:build linux && !android

package wayland

import (
	"context"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/lewtec/lewkit/x/release"
	"golang.org/x/sys/unix"
)

func (factory) CheckCompatibility(ctx context.Context) error {
	path, err := waylandSocket(ctx)
	if err != nil {
		return err
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%w: wayland socket: %v", driver.ErrIncompatible, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%w: wayland socket is a directory", driver.ErrIncompatible)
	}
	return nil
}

func waylandSocket(ctx context.Context) (string, error) {
	name := strings.TrimSpace(driver.GetEnv(ctx, "WAYLAND_DISPLAY"))
	if name == "" {
		return "", fmt.Errorf("%w: WAYLAND_DISPLAY not set", driver.ErrIncompatible)
	}
	if filepath.IsAbs(name) {
		return name, nil
	}
	dir := strings.TrimSpace(driver.GetEnv(ctx, "XDG_RUNTIME_DIR"))
	if dir == "" {
		return "", fmt.Errorf("%w: XDG_RUNTIME_DIR not set", driver.ErrIncompatible)
	}
	return filepath.Join(dir, name), nil
}

type global struct {
	name, version uint32
	iface         string
}

type wlwin struct {
	*window.Buffer
	mu    sync.Mutex
	blit  sync.Mutex
	want  window.WantSize
	conn  *conn
	ready chan struct{}

	registry   uint32
	compositor uint32
	shm        uint32
	wm         uint32
	seat       uint32
	surface    uint32
	xdgSurface uint32
	toplevel   uint32
	globals    []global

	pointerID  uint32
	keyboardID uint32
	point      image.Point
	buttons    int
	mods       window.Modifier

	format     uint32
	haveFormat bool
	pool       uint32
	poolFD     int
	poolW      int
	poolH      int
	mem        []byte
	slots      [2]shmSlot
	bgra       []byte
	closed     int32
}

func (opener) Open(ctx context.Context, cfg window.Config) (window.Window, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	width, height, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	path, err := waylandSocket(ctx)
	if err != nil {
		return nil, err
	}
	conn, err := dial(path)
	if err != nil {
		return nil, fmt.Errorf("wayland: %w", err)
	}
	buf := window.NewBuffer(width, height)
	buf.SetFramePeriod(cfg.Period)
	w := &wlwin{
		Buffer: buf,
		want:   window.WantSize{Width: width, Height: height},
		conn:   conn,
		ready:  make(chan struct{}),
		poolFD: -1,
	}
	if err := w.negotiate(ctx, cfg.Title); err != nil {
		w.Close()
		return nil, err
	}
	window.CloseWhenDone(ctx, w)
	return w, nil
}

func (w *wlwin) negotiate(ctx context.Context, title string) error {
	w.registry = w.conn.alloc(w.onRegistry)
	if err := w.conn.send(1, 1, pack(w.registry)); err != nil {
		return err
	}
	if err := w.conn.roundtrip(ctx); err != nil {
		return err
	}
	w.compositor = w.bind("wl_compositor", 4, nil)
	w.shm = w.bind("wl_shm", 1, func(uint32) func(uint16, []byte) { return w.onShm })
	w.wm = w.bind("xdg_wm_base", 4, func(id uint32) func(uint16, []byte) {
		return func(op uint16, payload []byte) { w.onWM(id, op, payload) }
	})
	w.seat = w.bind("wl_seat", 5, func(id uint32) func(uint16, []byte) {
		return func(op uint16, payload []byte) { w.onSeat(id, op, payload) }
	})
	if w.compositor == 0 || w.shm == 0 || w.wm == 0 {
		return fmt.Errorf("wayland: compositor, shm, or xdg_wm_base missing")
	}
	if err := w.conn.roundtrip(ctx); err != nil {
		return err
	}
	if !w.haveFormat {
		return fmt.Errorf("wayland: no argb8888 shm format")
	}
	w.surface = w.conn.alloc(nil)
	if err := w.conn.send(w.compositor, 0, pack(w.surface)); err != nil {
		return err
	}
	w.xdgSurface = w.conn.alloc(w.onXdgSurface)
	if err := w.conn.send(w.wm, 2, pack(w.xdgSurface, w.surface)); err != nil {
		return err
	}
	w.toplevel = w.conn.alloc(w.onToplevel)
	if err := w.conn.send(w.xdgSurface, 1, pack(w.toplevel)); err != nil {
		return err
	}
	if title != "" {
		if err := w.conn.send(w.toplevel, 2, pack(title)); err != nil {
			return err
		}
	}
	if err := w.conn.send(w.toplevel, 3, pack(appID())); err != nil {
		return err
	}
	if err := w.conn.send(w.toplevel, 8, pack(int32(1), int32(1))); err != nil {
		return err
	}
	if err := w.conn.send(w.surface, 6, nil); err != nil {
		return err
	}
	select {
	case <-w.ready:
	case <-ctx.Done():
		return ctx.Err()
	case <-w.conn.dead:
		return w.conn.fatal()
	}
	if _, err := w.Buffer.EnsureSize(w.Size()); err != nil {
		return err
	}
	return w.put()
}

func (w *wlwin) bind(iface string, max uint32, fn func(uint32) func(uint16, []byte)) uint32 {
	for _, g := range w.globals {
		if g.iface != iface || g.version < 1 {
			continue
		}
		version := g.version
		if version > max {
			version = max
		}
		id := w.conn.alloc(nil)
		if fn != nil {
			w.conn.set(id, fn(id))
		}
		if err := w.conn.send(w.registry, 0, pack(g.name, iface, version, id)); err != nil {
			return 0
		}
		return id
	}
	return 0
}

func (w *wlwin) onRegistry(op uint16, payload []byte) {
	if op != 0 {
		return
	}
	name, payload := takeU32(payload)
	iface, payload := takeString(payload)
	version, _ := takeU32(payload)
	if iface == "" || version == 0 {
		return
	}
	w.globals = append(w.globals, global{name: name, version: version, iface: iface})
}

func (w *wlwin) onShm(op uint16, payload []byte) {
	if op != 0 {
		return
	}
	format, _ := takeU32(payload)
	if format == 0 {
		w.format = 0
		w.haveFormat = true
		return
	}
	if format == 1 && !w.haveFormat {
		w.format = 1
		w.haveFormat = true
	}
}

func (w *wlwin) onWM(id uint32, op uint16, payload []byte) {
	if op != 0 {
		return
	}
	serial, _ := takeU32(payload)
	_ = w.conn.send(id, 3, pack(serial))
}

func (w *wlwin) onToplevel(op uint16, payload []byte) {
	switch op {
	case 0:
		width, payload := takeI32(payload)
		height, _ := takeI32(payload)
		if width > 0 && height > 0 {
			window.SetWant(w.Buffer, &w.mu, &w.want, int(width), int(height))
		}
	case 1:
		_ = w.Close()
	}
}

func (w *wlwin) onXdgSurface(op uint16, payload []byte) {
	if op != 0 {
		return
	}
	serial, _ := takeU32(payload)
	_ = w.conn.send(w.xdgSurface, 4, pack(serial))
	w.Buffer.Emit(window.Expose{})
	select {
	case <-w.ready:
	default:
		close(w.ready)
	}
}

func appID() string {
	id, err := release.AppID()
	if err != nil || id == "" {
		return release.Name()
	}
	return id
}

func (w *wlwin) Size() image.Point {
	return window.HostSize(w.Buffer, &w.mu, &w.want)
}

func (w *wlwin) Frame() *image.RGBA {
	return window.HostFrame(w.Buffer, w.Size())
}

func (w *wlwin) Draw() error {
	return window.SwapBlit(w.Buffer, w.put)
}

func (w *wlwin) Resize(size image.Point) error {
	w.mu.Lock()
	w.want.Width, w.want.Height = size.X, size.Y
	w.mu.Unlock()
	if err := w.Buffer.Resize(size); err != nil {
		return err
	}
	if atomic.LoadInt32(&w.closed) != 0 || w.conn == nil || w.surface == 0 {
		return window.ErrClosed
	}
	return nil
}

func (w *wlwin) Close() error {
	if w == nil {
		return nil
	}
	if !atomic.CompareAndSwapInt32(&w.closed, 0, 1) {
		return nil
	}
	w.blit.Lock()
	mem := w.mem
	fd := w.poolFD
	w.mem = nil
	w.poolFD = -1
	conn := w.conn
	w.blit.Unlock()
	if w.Buffer != nil {
		_ = w.Buffer.Close()
	}
	if conn != nil {
		conn.close()
	}
	if mem != nil {
		_ = unix.Munmap(mem)
	}
	if fd >= 0 {
		_ = unix.Close(fd)
	}
	return nil
}
