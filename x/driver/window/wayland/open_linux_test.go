//go:build linux && !android

package wayland

import (
	"context"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver"
	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestCheckNeedsWayland(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
	require.ErrorContains(t, err, "WAYLAND_DISPLAY")
}

func TestCheckMissingSocket(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "/tmp/lewkit-wayland-missing")
	err := factory{}.CheckCompatibility(t.Context())
	require.ErrorIs(t, err, driver.ErrIncompatible)
}

func TestPackStringPads(t *testing.T) {
	raw := pack("ab")
	require.Equal(t, 8, len(raw))
	require.Equal(t, byte(3), raw[0])
	require.Equal(t, byte('a'), raw[4])
	require.Equal(t, byte('b'), raw[5])
	require.Equal(t, byte(0), raw[6])
}

func TestEvdevRune(t *testing.T) {
	require.Equal(t, 'a', evdevRune(30, false))
	require.Equal(t, 'A', evdevRune(30, true))
	require.Equal(t, '+', evdevRune(13, true))
	require.Equal(t, '\n', evdevRune(28, false))
	require.Equal(t, rune(0), evdevRune(999, false))
}

func TestPointerButton(t *testing.T) {
	number, mask := pointerButton(0x110)
	require.Equal(t, 1, number)
	require.Equal(t, window.ButtonLeft, mask)
	number, _ = pointerButton(0x111)
	require.Equal(t, 3, number)
	number, _ = pointerButton(0x112)
	require.Equal(t, 2, number)
}

func TestOpenNilContextPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil context returned")
		}
	}()
	_, _ = opener{}.Open(nil, window.Config{})
}

func TestOpenAgainstFakeCompositor(t *testing.T) {
	dir, err := os.MkdirTemp("/tmp", "wl")
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	path := filepath.Join(dir, "s")
	lfd, err := listenSocket(path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = unix.Close(lfd) })

	mapped := make(chan []byte, 1)
	go acceptOne(lfd, mapped)

	t.Setenv("WAYLAND_DISPLAY", path)
	require.NoError(t, factory{}.CheckCompatibility(t.Context()))
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	win, err := opener{}.Open(ctx, window.Config{Title: "Triangle", Width: 800, Height: 600})
	require.NoError(t, err)
	t.Cleanup(func() { _ = win.Close() })
	require.Equal(t, image.Pt(640, 480), win.Size())

	win.Frame().SetRGBA(1, 2, color.RGBA{R: 10, G: 20, B: 30, A: 255})
	require.NoError(t, win.Draw())

	var mem []byte
	select {
	case mem = <-mapped:
	case <-ctx.Done():
		t.Fatal("compositor did not map the pool")
	}
	want := []byte{30, 20, 10, 255}
	found := false
	for i := 0; i+4 <= len(mem); i++ {
		if mem[i] == want[0] && mem[i+1] == want[1] && mem[i+2] == want[2] && mem[i+3] == want[3] {
			found = true
			break
		}
	}
	require.True(t, found, "painted pixel missing from shm")
}

func listenSocket(path string) (int, error) {
	_ = unix.Unlink(path)
	fd, err := unix.Socket(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return -1, err
	}
	if err := unix.Bind(fd, &unix.SockaddrUnix{Name: path}); err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	if err := unix.Listen(fd, 1); err != nil {
		_ = unix.Close(fd)
		return -1, err
	}
	return fd, nil
}

func acceptOne(lfd int, mapped chan<- []byte) {
	fd, _, err := unix.Accept(lfd)
	if err != nil {
		return
	}
	defer unix.Close(fd)
	s := &stream{fd: fd, scratch: make([]byte, 8192), oob: make([]byte, unix.CmsgSpace(256))}
	var registry, surface, xdgSurface, toplevel, shm, comp, wm uint32
	configured := false
	for {
		id, op, payload, err := s.next()
		if err != nil {
			return
		}
		switch {
		case id == 1 && op == 1:
			registry, _ = takeU32(payload)
			for _, g := range []struct {
				name    uint32
				iface   string
				version uint32
			}{
				{1, "wl_compositor", 4},
				{2, "wl_shm", 1},
				{3, "xdg_wm_base", 4},
				{4, "wl_seat", 5},
			} {
				_ = writeMsg(fd, packEvent(registry, 0, pack(g.name, g.iface, g.version)), nil)
			}
		case id == 1 && op == 0:
			cb, _ := takeU32(payload)
			_ = writeMsg(fd, packEvent(cb, 0, pack(uint32(0))), nil)
			_ = writeMsg(fd, packEvent(1, 1, pack(cb)), nil)
		case id == registry && op == 0:
			_, payload = takeU32(payload)
			iface, payload := takeString(payload)
			_, payload = takeU32(payload)
			newID, _ := takeU32(payload)
			switch iface {
			case "wl_compositor":
				comp = newID
			case "wl_shm":
				shm = newID
				_ = writeMsg(fd, packEvent(shm, 0, pack(uint32(0))), nil)
			case "xdg_wm_base":
				wm = newID
			case "wl_seat":
				_ = writeMsg(fd, packEvent(newID, 0, pack(uint32(3))), nil)
			}
		case id == comp && op == 0:
			surface, _ = takeU32(payload)
		case id == wm && op == 2:
			xdgSurface, _ = takeU32(payload)
		case id == xdgSurface && op == 1:
			toplevel, _ = takeU32(payload)
		case id == surface && op == 6 && !configured && toplevel != 0 && xdgSurface != 0:
			configured = true
			body := pack(int32(640), int32(480), uint32(0))
			_ = writeMsg(fd, packEvent(toplevel, 0, body), nil)
			_ = writeMsg(fd, packEvent(xdgSurface, 0, pack(uint32(7))), nil)
		case id == shm && op == 0:
			poolFD := -1
			if len(s.fds) > 0 {
				poolFD = s.fds[0]
				s.fds = s.fds[1:]
			}
			if poolFD >= 0 {
				size := 640 * 480 * 4 * 2
				mem, err := unix.Mmap(poolFD, 0, size, unix.PROT_READ, unix.MAP_SHARED)
				if err == nil {
					select {
					case mapped <- mem:
					default:
					}
				}
			}
		}
	}
}

func packEvent(id uint32, opcode uint16, body []byte) []byte {
	size := 8 + len(body)
	msg := make([]byte, size)
	msg[0] = byte(id)
	msg[1] = byte(id >> 8)
	msg[2] = byte(id >> 16)
	msg[3] = byte(id >> 24)
	hdr := uint32(size)<<16 | uint32(opcode)
	msg[4] = byte(hdr)
	msg[5] = byte(hdr >> 8)
	msg[6] = byte(hdr >> 16)
	msg[7] = byte(hdr >> 24)
	copy(msg[8:], body)
	return msg
}
