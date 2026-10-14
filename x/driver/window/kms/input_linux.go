//go:build linux

package kms

import (
	"encoding/binary"
	"image"
	"os"
	"path/filepath"
	"time"
	"unsafe"

	"github.com/lewtec/lewkit/x/driver/window"
	"golang.org/x/sys/unix"
)

const (
	evSyn = 0
	evKey = 1
	evRel = 2
	evAbs = 3

	synReport = 0
	relX      = 0
	relY      = 1
	relWheel  = 8
	absX      = 0
	absY      = 1

	keyLeftShift  = 42
	keyRightShift = 54
	keyLeftCtrl   = 29
	keyRightCtrl  = 97
	keyLeftAlt    = 56
	keyRightAlt   = 100
	keyLeftMeta   = 125
	keyRightMeta  = 126

	btnLeft   = 0x110
	btnRight  = 0x111
	btnMiddle = 0x112
	btnTouch  = 0x14a
)

// inputEventLen is timeval (two 64-bit words) plus type, code, and value.
const inputEventLen = 24

type evDev struct {
	path         string
	fd           int
	abs          bool
	minX, maxX   int32
	minY, maxY   int32
	x, y         int
	buttons      int
	moved        bool
	pointerDirty bool
}

func (w *fbWin) loopInput() {
	open := map[string]*evDev{}
	defer func() {
		for _, dev := range open {
			_ = unix.Close(dev.fd)
		}
	}()
	for {
		w.refreshInputs(open)
		if len(open) == 0 {
			timer := time.NewTimer(200 * time.Millisecond)
			select {
			case <-w.stop:
				timer.Stop()
				return
			case <-timer.C:
			}
			continue
		}
		pf := make([]unix.PollFd, 0, len(open))
		devs := make([]*evDev, 0, len(open))
		for _, dev := range open {
			pf = append(pf, unix.PollFd{Fd: int32(dev.fd), Events: unix.POLLIN})
			devs = append(devs, dev)
		}
		_, err := unix.Poll(pf, 200)
		select {
		case <-w.stop:
			return
		default:
		}
		if err != nil && err != unix.EINTR {
			continue
		}
		for i, slot := range pf {
			if slot.Revents == 0 {
				continue
			}
			if slot.Revents&(unix.POLLHUP|unix.POLLERR|unix.POLLNVAL) != 0 && slot.Revents&unix.POLLIN == 0 {
				_ = unix.Close(devs[i].fd)
				delete(open, devs[i].path)
				continue
			}
			w.readDev(devs[i])
		}
	}
}

func (w *fbWin) refreshInputs(open map[string]*evDev) {
	entries, err := os.ReadDir("/dev/input")
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if len(name) < 6 || name[:5] != "event" {
			continue
		}
		path := filepath.Join("/dev/input", name)
		if _, ok := open[path]; ok {
			continue
		}
		fd, err := unix.Open(path, unix.O_RDONLY|unix.O_NONBLOCK|unix.O_CLOEXEC, 0)
		if err != nil {
			continue
		}
		dev := &evDev{path: path, fd: fd}
		if min, max, ok := absRange(fd, absX); ok {
			dev.abs = true
			dev.minX, dev.maxX = min, max
		}
		if min, max, ok := absRange(fd, absY); ok {
			dev.abs = true
			dev.minY, dev.maxY = min, max
		}
		open[path] = dev
	}
}

func (w *fbWin) readDev(dev *evDev) {
	buf := make([]byte, inputEventLen*16)
	for {
		n, err := unix.Read(dev.fd, buf)
		if n <= 0 || err != nil {
			return
		}
		for off := 0; off+inputEventLen <= n; off += inputEventLen {
			typ := binary.LittleEndian.Uint16(buf[off+16:])
			code := binary.LittleEndian.Uint16(buf[off+18:])
			value := int32(binary.LittleEndian.Uint32(buf[off+20:]))
			w.apply(dev, typ, code, value)
		}
	}
}

func (w *fbWin) apply(dev *evDev, typ, code uint16, value int32) {
	switch typ {
	case evKey:
		w.applyKey(dev, code, value)
	case evRel:
		switch code {
		case relX:
			dev.x += int(value)
			dev.moved = true
		case relY:
			dev.y += int(value)
			dev.moved = true
		case relWheel:
			w.Emit(window.Scroll{Pos: image.Pt(dev.x, dev.y), Delta: image.Pt(0, int(-value))})
		}
	case evAbs:
		switch code {
		case absX:
			dev.x = scaleAbs(value, dev.minX, dev.maxX, w.width)
			dev.moved = true
		case absY:
			dev.y = scaleAbs(value, dev.minY, dev.maxY, w.height)
			dev.moved = true
		}
	case evSyn:
		if code != synReport || !dev.pointerDirty && !dev.moved {
			return
		}
		dev.x = clamp(dev.x, 0, w.width-1)
		dev.y = clamp(dev.y, 0, w.height-1)
		dev.moved = false
		dev.pointerDirty = false
		w.Emit(window.Pointer{Pos: image.Pt(dev.x, dev.y), Buttons: dev.buttons})
	}
}

func (w *fbWin) applyKey(dev *evDev, code uint16, value int32) {
	down := value != 0
	if mod, ok := keyMod(code); ok {
		if down {
			w.mods |= mod
		} else {
			w.mods &^= mod
		}
	}
	if button, ok := keyButton(code); ok {
		if down {
			dev.buttons |= button
		} else {
			dev.buttons &^= button
		}
		dev.pointerDirty = true
		edge := 0
		pressed := false
		switch button {
		case window.ButtonLeft:
			edge, pressed = 1, down
		case window.ButtonRight:
			edge, pressed = 2, down
		case window.ButtonMiddle:
			edge, pressed = 3, down
		}
		w.Emit(window.Pointer{Pos: image.Pt(dev.x, dev.y), Button: edge, Pressed: pressed, Buttons: dev.buttons})
		return
	}
	if code >= 0x100 {
		return
	}
	w.Emit(window.Key{
		Rune:    window.EvdevRune(uint32(code), w.mods&window.ModShift != 0),
		Code:    uint32(code),
		Pressed: down,
		Repeat:  value == 2,
		Mod:     w.mods,
	})
}

func keyMod(code uint16) (window.Modifier, bool) {
	switch code {
	case keyLeftShift, keyRightShift:
		return window.ModShift, true
	case keyLeftCtrl, keyRightCtrl:
		return window.ModCtrl, true
	case keyLeftAlt, keyRightAlt:
		return window.ModAlt, true
	case keyLeftMeta, keyRightMeta:
		return window.ModSuper, true
	default:
		return 0, false
	}
}

func keyButton(code uint16) (int, bool) {
	switch code {
	case btnLeft, btnTouch:
		return window.ButtonLeft, true
	case btnRight:
		return window.ButtonRight, true
	case btnMiddle:
		return window.ButtonMiddle, true
	default:
		return 0, false
	}
}

func scaleAbs(value, min, max int32, span int) int {
	if span <= 1 || max <= min {
		return 0
	}
	if value < min {
		value = min
	}
	if value > max {
		value = max
	}
	return int((int64(value-min) * int64(span-1)) / int64(max-min))
}

func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func absRange(fd int, axis uint16) (min, max int32, ok bool) {
	var info [6]int32
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(fd), evIocGAbs(axis), uintptr(unsafe.Pointer(&info[0])))
	if errno != 0 || info[2] <= info[1] {
		return 0, 0, false
	}
	return info[1], info[2], true
}

func evIocGAbs(axis uint16) uintptr {
	const (
		iocRead = 2
		size    = 24
		nr      = 0x40
	)
	return uintptr(iocRead<<30 | size<<16 | int('E')<<8 | int(nr+axis))
}
