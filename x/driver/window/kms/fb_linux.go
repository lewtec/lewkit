//go:build linux

package kms

import (
	"context"
	"fmt"
	"image"
	"os"
	"sync"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"golang.org/x/sys/unix"
)

const (
	// Historical Linux fb ioctls, not the _IOR encoding.
	fbiogetVscreeninfo = 0x4600
	fbioputVscreeninfo = 0x4601
	fbiogetFscreeninfo = 0x4602
	fbiopanDisplay     = 0x4606
)

type fbBitfield struct {
	Offset   uint32
	Length   uint32
	MsbRight uint32
}

type fbVar struct {
	Xres, Yres, XresVirtual, YresVirtual uint32
	Xoffset, Yoffset                     uint32
	BitsPerPixel, Grayscale              uint32
	Red, Green, Blue, Transp             fbBitfield
	Nonstd, Activate, Height, Width      uint32
	AccelFlags, Pixclock                 uint32
	LeftMargin, RightMargin              uint32
	UpperMargin, LowerMargin             uint32
	HsyncLen, VsyncLen                   uint32
	Sync, Vmode, Rotate, Colorspace      uint32
	Reserved                             [4]uint32
}

type fbFix struct {
	ID           [16]byte
	SmemStart    uint64
	SmemLen      uint32
	Type         uint32
	TypeAux      uint32
	Visual       uint32
	Xpanstep     uint16
	Ypanstep     uint16
	Ywrapstep    uint16
	_            uint16
	LineLength   uint32
	_            uint32
	MmioStart    uint64
	MmioLen      uint32
	Accel        uint32
	Capabilities uint16
	Reserved     [2]uint16
	_            uint16
}

func (factory) CheckCompatibility(context.Context) error {
	f, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("%w: /dev/fb0: %v", driver.ErrIncompatible, err)
	}
	return f.Close()
}

func (opener) Open(_ context.Context, cfg window.Config) (window.Window, error) {
	if _, _, err := cfg.Size(); err != nil {
		return nil, err
	}
	f, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0)
	if err != nil {
		return nil, err
	}
	screen, err := mapFB(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	// The framebuffer mode is the window. The requested size is a hint.
	buf := window.NewBuffer(screen.width, screen.height)
	buf.SetFramePeriod(cfg.Period)
	win := &fbWin{Buffer: buf, f: f, mem: screen.mem, width: screen.width, height: screen.height, stride: screen.stride, bpp: screen.bpp, red: screen.red, green: screen.green, blue: screen.blue, stop: make(chan struct{})}
	go win.loopInput()
	return win, nil
}

type screen struct {
	mem                   []byte
	width, height, stride int
	bpp                   uint32
	red, green, blue      uint32
}

func mapFB(f *os.File) (screen, error) {
	v, err := prepareVar(f)
	if err != nil {
		return screen{}, err
	}
	var fix fbFix
	if err := ioctl(f.Fd(), fbiogetFscreeninfo, unsafe.Pointer(&fix)); err != nil {
		return screen{}, err
	}
	// guestDrew treats this line as proof the kms driver owns the page.
	fmt.Fprintf(os.Stderr, "kms: %dx%d stride %d bpp %d\n", v.Xres, v.Yres, fix.LineLength, v.BitsPerPixel)
	if v.Xres == 0 || v.Yres == 0 || fix.LineLength == 0 {
		return screen{}, fmt.Errorf("kms: empty framebuffer")
	}
	if v.BitsPerPixel != 32 {
		return screen{}, fmt.Errorf("kms: framebuffer is %d-bit", v.BitsPerPixel)
	}
	size := int(fix.SmemLen)
	need := int(fix.LineLength) * int(v.Yres)
	if size < need {
		size = need
	}
	mem, err := unix.Mmap(int(f.Fd()), 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_SHARED)
	if err != nil {
		return screen{}, err
	}
	return screen{
		mem: mem, width: int(v.Xres), height: int(v.Yres), stride: int(fix.LineLength),
		bpp: v.BitsPerPixel, red: v.Red.Offset, green: v.Green.Offset, blue: v.Blue.Offset,
	}, nil
}

// prepareVar asks for a 32-bit mode when the current one cannot hold a window.
// The 720×400 page is the VGA text mode QEMU serves before modeset.
func prepareVar(f *os.File) (fbVar, error) {
	var v fbVar
	if err := ioctl(f.Fd(), fbiogetVscreeninfo, unsafe.Pointer(&v)); err != nil {
		return v, err
	}
	if usableMode(v) {
		return v, nil
	}
	orig := v
	v.Xres, v.Yres = 1024, 768
	v.XresVirtual, v.YresVirtual = 1024, 768
	v.BitsPerPixel = 32
	v.Activate = 0
	v.Pixclock = 0
	v.Red = fbBitfield{Offset: 16, Length: 8}
	v.Green = fbBitfield{Offset: 8, Length: 8}
	v.Blue = fbBitfield{Offset: 0, Length: 8}
	v.Transp = fbBitfield{Offset: 24, Length: 8}
	if err := ioctl(f.Fd(), fbioputVscreeninfo, unsafe.Pointer(&v)); err != nil {
		// ramfb and simpledrm often refuse a modeset. A 32-bit page is still drawable.
		if orig.BitsPerPixel == 32 && orig.Xres >= 640 && orig.Yres >= 400 {
			return orig, nil
		}
		return v, fmt.Errorf("kms: set mode: %w", err)
	}
	if err := ioctl(f.Fd(), fbiogetVscreeninfo, unsafe.Pointer(&v)); err != nil {
		return v, err
	}
	return v, nil
}

func usableMode(v fbVar) bool {
	return v.BitsPerPixel == 32 && v.Xres >= 640 && v.Yres >= 480 && !(v.Xres == 720 && v.Yres == 400)
}

func ioctl(fd uintptr, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

type fbWin struct {
	*window.Buffer
	f                     *os.File
	mem                   []byte
	width, height, stride int
	bpp, red, green, blue uint32
	stage                 []byte
	stop                  chan struct{}
	stopOnce              sync.Once
	mods                  window.Modifier
}

func (w *fbWin) Draw() error {
	return window.SwapBlit(w.Buffer, w.blit)
}

func (w *fbWin) blit() error {
	var src *image.RGBA
	w.WithFront(func(front *image.RGBA) { src = front })
	if src == nil || w.f == nil {
		return nil
	}
	need := w.stride * w.height
	if need <= 0 {
		return nil
	}
	if cap(w.stage) < need {
		w.stage = make([]byte, need)
	}
	stage := w.stage[:need]
	clear(stage)
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	for y := 0; y < sh && y < w.height; y++ {
		row := src.Pix[y*src.Stride:]
		dst := stage[y*w.stride:]
		for x := 0; x < sw && x < w.width; x++ {
			i := x * 4
			if i+3 >= len(row) || i+3 >= len(dst) {
				break
			}
			put32(dst[i:], row[i], row[i+1], row[i+2], w.red, w.green, w.blue)
		}
	}
	// mmap writes stay in the DRM shadow. write() is fb_write, which schedules
	// the damage blit onto the scanout. A pan commits that plane.
	if err := pwriteAll(int(w.f.Fd()), stage); err != nil {
		return fmt.Errorf("kms: write fb: %w", err)
	}
	if w.mem != nil && len(w.mem) >= len(stage) {
		copy(w.mem, stage)
		_ = unix.Msync(w.mem, unix.MS_SYNC)
	}
	var v fbVar
	if ioctl(w.f.Fd(), fbiogetVscreeninfo, unsafe.Pointer(&v)) == nil {
		_ = ioctl(w.f.Fd(), fbiopanDisplay, unsafe.Pointer(&v))
	}
	return nil
}

func pwriteAll(fd int, p []byte) error {
	var off int64
	for len(p) > 0 {
		n, err := unix.Pwrite(fd, p, off)
		if n > 0 {
			p = p[n:]
			off += int64(n)
		}
		if err != nil {
			return err
		}
		if n == 0 {
			return fmt.Errorf("short write at %d", off)
		}
	}
	return nil
}

func put32(dst []byte, r, g, b byte, red, green, blue uint32) {
	var pix uint32
	pix |= uint32(r) << red
	pix |= uint32(g) << green
	pix |= uint32(b) << blue
	dst[0] = byte(pix)
	dst[1] = byte(pix >> 8)
	dst[2] = byte(pix >> 16)
	dst[3] = byte(pix >> 24)
}

func (w *fbWin) Close() error {
	if w.stop != nil {
		w.stopOnce.Do(func() { close(w.stop) })
	}
	err := w.Buffer.Close()
	if w.mem != nil {
		_ = unix.Munmap(w.mem)
		w.mem = nil
	}
	if w.f != nil {
		_ = w.f.Close()
		w.f = nil
	}
	return err
}
