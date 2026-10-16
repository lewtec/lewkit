package uki

import (
	"bytes"
	"context"
	"debug/elf"
	"encoding/binary"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	isofs "github.com/lewtec/lewkit/x/fs/iso"
	"github.com/lewtec/lewkit/x/test"

	"github.com/stretchr/testify/require"
)

func TestGuestArch(t *testing.T) {
	require.True(t, Guest("linux_uki"))
	require.True(t, Guest("linux_iso"))
	require.False(t, Guest("uki"))
	require.False(t, Guest("linux"))
	arch, err := AlpineArch("amd64")
	require.NoError(t, err)
	require.Equal(t, "x86_64", arch)
	arch, err = AlpineArch("arm64")
	require.NoError(t, err)
	require.Equal(t, "aarch64", arch)
	_, err = AlpineArch("386")
	require.ErrorIs(t, err, ErrArch)
	url, err := kernelURL("amd64")
	require.NoError(t, err)
	require.Contains(t, url, "/v3.24/main/x86_64/linux-virt-6.18.56-r0.apk")
	url, err = kernelURL("arm64")
	require.NoError(t, err)
	require.Contains(t, url, "/v3.24/main/aarch64/linux-virt-6.18.56-r0.apk")
	url, err = stubURL("amd64")
	require.NoError(t, err)
	require.Contains(t, url, "systemd-boot-efi_257.13-1~deb13u1_amd64.deb")
	url, err = stubURL("arm64")
	require.NoError(t, err)
	require.Contains(t, url, "systemd-boot-efi_257.13-1~deb13u1_arm64.deb")
	_, err = stubURL("386")
	require.ErrorIs(t, err, ErrArch)
	err = Build(t.Context(), Options{GOOS: "uki", GOARCH: "amd64", Out: filepath.Join(t.TempDir(), "x"), Dir: t.TempDir()})
	require.Error(t, err)
}

func TestWriteISO(t *testing.T) {
	payload := append([]byte("MZ\x00\n"), bytes.Repeat([]byte{0x11}, 1000)...)
	iso, err := writeISO("amd64", payload)
	require.NoError(t, err)
	require.Equal(t, "CD001", string(iso[0x8001:0x8006]))
	got, err := isofs.ReadFile(iso, "EFI/BOOT/BOOTX64.EFI")
	require.NoError(t, err)
	require.Equal(t, payload, got)
	arch, err := isoBootArch(iso)
	require.NoError(t, err)
	require.Equal(t, "amd64", arch)

	iso, err = writeISO("arm64", payload)
	require.NoError(t, err)
	got, err = isofs.ReadFile(iso, "EFI/BOOT/BOOTAA64.EFI")
	require.NoError(t, err)
	require.Equal(t, payload, got)
	arch, err = isoBootArch(iso)
	require.NoError(t, err)
	require.Equal(t, "arm64", arch)

	_, err = writeISO("386", payload)
	require.ErrorIs(t, err, ErrArch)
	_, err = writeISO("amd64", nil)
	require.Error(t, err)
}

func TestPERoundTrip(t *testing.T) {
	stub := minimalPE(t, 0x8664)
	got, err := assembleUKI(stub, []byte("MZkernel"), []byte("070701initrd"), []byte(guestCmdline), []byte("6.18.56-0-virt"), osRelease("0.1.0"))
	require.NoError(t, err)
	arch, err := imageArch(got)
	require.NoError(t, err)
	require.Equal(t, "amd64", arch)
	linux, err := peSection(got, ".linux")
	require.NoError(t, err)
	require.Equal(t, []byte("MZkernel"), linux)
	initrd, err := peSection(got, ".initrd")
	require.NoError(t, err)
	require.Equal(t, []byte("070701initrd"), initrd)
	cmd, err := peSection(got, ".cmdline")
	require.NoError(t, err)
	require.Equal(t, guestCmdline+"\x00", string(cmd))
	uname, err := peSection(got, ".uname")
	require.NoError(t, err)
	require.Equal(t, "6.18.56-0-virt\x00", string(uname))
	osrel, err := peSection(got, ".osrel")
	require.NoError(t, err)
	require.Contains(t, string(osrel), "VERSION_ID=0.1.0\n")
	_, err = assembleUKI(stub, nil, []byte("x"), nil, nil, nil)
	require.Error(t, err)
	arm := minimalPE(t, 0xaa64)
	arch, err = imageArch(arm)
	require.NoError(t, err)
	require.Equal(t, "arm64", arch)
}

func minimalPE(t *testing.T, machine uint16) []byte {
	t.Helper()
	const (
		peoff   = 0x80
		optSize = 240
		headers = 0x400
		rawSize = 0x200
	)
	buf := make([]byte, headers+rawSize)
	buf[0], buf[1] = 'M', 'Z'
	binary.LittleEndian.PutUint32(buf[0x3c:], peoff)
	copy(buf[peoff:], []byte("PE\x00\x00"))
	coff := peoff + 4
	binary.LittleEndian.PutUint16(buf[coff:], machine)
	binary.LittleEndian.PutUint16(buf[coff+2:], 1)
	binary.LittleEndian.PutUint16(buf[coff+16:], optSize)
	opt := coff + 20
	binary.LittleEndian.PutUint16(buf[opt:], pe32Plus)
	binary.LittleEndian.PutUint32(buf[opt+32:], 0x1000)
	binary.LittleEndian.PutUint32(buf[opt+36:], 0x200)
	binary.LittleEndian.PutUint32(buf[opt+56:], 0x2000)
	binary.LittleEndian.PutUint32(buf[opt+60:], headers)
	binary.LittleEndian.PutUint16(buf[opt+68:], 10)
	binary.LittleEndian.PutUint32(buf[opt+108:], 16)
	table := opt + optSize
	copy(buf[table:], []byte(".text"))
	binary.LittleEndian.PutUint32(buf[table+8:], 0x10)
	binary.LittleEndian.PutUint32(buf[table+12:], 0x1000)
	binary.LittleEndian.PutUint32(buf[table+16:], rawSize)
	binary.LittleEndian.PutUint32(buf[table+20:], headers)
	binary.LittleEndian.PutUint32(buf[table+36:], 0x60000020)
	return buf
}

func TestReadAlpineKernel(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	apk, err := fetchKernel(ctx, "amd64")
	require.NoError(t, err)
	image, err := readKernel(apk, true)
	require.NoError(t, err)
	require.Greater(t, len(image.kernel), 1<<20)
	require.Equal(t, "6.18.56-0-virt", image.version)
	require.Less(t, stemIndex(image.order, "drm"), stemIndex(image.order, "bochs"))
	require.Less(t, stemIndex(image.order, "evdev"), stemIndex(image.order, "usbhid"))
	for _, stem := range []string{"simpledrm", "virtio-gpu", "bochs", "evdev", "virtio-input", "usbhid", "hid-generic", "xhci-pci", "ehci-pci", "uhci-hcd", "virtio-net"} {
		require.NotEqual(t, -1, stemIndex(image.order, stem), stem)
	}
}

func stemIndex(order []string, stem string) int {
	for i, name := range order {
		if moduleStem(name) == stem {
			return i
		}
	}
	return -1
}

func TestStubPacks(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Minute)
	defer cancel()
	for _, goarch := range []string{"amd64", "arm64"} {
		stub, err := fetchStub(ctx, goarch)
		require.NoError(t, err)
		require.Equal(t, stubSHA[goarch], sha256Hex(stub))
		got, err := assembleUKI(stub, []byte("MZkernel"), []byte("070701initrd"), []byte(guestCmdline), []byte("6.18.56-0-virt"), osRelease("0.1.0"))
		require.NoError(t, err)
		arch, err := imageArch(got)
		require.NoError(t, err)
		require.Equal(t, goarch, arch)
		linux, err := peSection(got, ".linux")
		require.NoError(t, err)
		require.Equal(t, []byte("MZkernel"), linux)
	}
}

func TestBuildDownloadsKernel(t *testing.T) {
	if testing.Short() {
		test.Skip(t, "builds a guest")
	}
	out := filepath.Join(t.TempDir(), "app.efi")
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     writeTiny(t),
		Out:     out,
		AppID:   "br.tec.lew.ukiapp",
		Version: "0.1.0",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	arch, err := imageArch(raw)
	require.NoError(t, err)
	require.Equal(t, "amd64", arch)
	linux, err := peSection(raw, ".linux")
	require.NoError(t, err)
	require.Greater(t, len(linux), 1<<20)
	require.Equal(t, []byte{'M', 'Z'}, linux[:2])
	initrd, err := peSection(raw, ".initrd")
	require.NoError(t, err)
	require.True(t, bytes.HasPrefix(initrd, []byte("070701")))
	cmd, err := peSection(raw, ".cmdline")
	require.NoError(t, err)
	require.Equal(t, guestCmdline+"\x00", string(cmd))
	uname, err := peSection(raw, ".uname")
	require.NoError(t, err)
	require.Equal(t, "6.18.56-0-virt\x00", string(uname))
	_, err = os.Stat(filepath.Join(out, "vmlinuz"))
	require.Error(t, err)
}

func writeTiny(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/ukiapp\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"uki app\") }\n"), 0o644))
	return dir
}

func writePaint(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/ukiapp\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(paintSrc), 0o644))
	return dir
}

const paintSrc = `package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

type bitfield struct{ Offset, Length, MsbRight uint32 }

type fbVar struct {
	Xres, Yres, XresVirtual, YresVirtual uint32
	Xoffset, Yoffset                     uint32
	BitsPerPixel, Grayscale              uint32
	Red, Green, Blue, Transp             bitfield
	Nonstd, Activate, Height, Width      uint32
	AccelFlags, Pixclock                 uint32
	LeftMargin, RightMargin              uint32
	UpperMargin, LowerMargin             uint32
	HsyncLen, VsyncLen                   uint32
	Sync, Vmode, Rotate, Colorspace      uint32
	Reserved                             [4]uint32
}

type fbFix struct {
	ID                             [16]byte
	SmemStart                      uint64
	SmemLen                        uint32
	Type, TypeAux, Visual          uint32
	Xpanstep, Ypanstep, Ywrapstep  uint16
	_                              uint16
	LineLength                     uint32
	_                              uint32
	MmioStart                      uint64
	MmioLen, Accel                 uint32
	Capabilities                   uint16
	Reserved                       [2]uint16
	_                              uint16
}

func ioctl(fd, req uintptr, arg unsafe.Pointer) error {
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, fd, req, uintptr(arg))
	if errno != 0 {
		return errno
	}
	return nil
}

func main() {
	f, err := os.OpenFile("/dev/fb0", os.O_RDWR, 0)
	if err != nil {
		fmt.Println("paint: open", err)
		select {}
	}
	var v fbVar
	if err := ioctl(f.Fd(), 0x4600, unsafe.Pointer(&v)); err != nil {
		fmt.Println("paint: get", err)
		select {}
	}
	var fix fbFix
	if err := ioctl(f.Fd(), 0x4602, unsafe.Pointer(&fix)); err != nil {
		fmt.Println("paint: fix", err)
		select {}
	}
	fmt.Printf("kms: %dx%d stride %d bpp %d smem %d\n", v.Xres, v.Yres, fix.LineLength, v.BitsPerPixel, fix.SmemLen)
	w, h, stride := int(v.Xres), int(v.Yres), int(fix.LineLength)
	if w < 16 || h < 16 || stride < w*4 {
		fmt.Println("paint: bad mode")
		select {}
	}
	fmt.Println("paint: mmaping", stride*h)
	mem, merr := syscall.Mmap(int(f.Fd()), 0, stride*h, syscall.PROT_READ|syscall.PROT_WRITE, syscall.MAP_SHARED)
	fmt.Println("paint: mmap", len(mem), merr)
	buf := make([]byte, stride*h)
	for y := 0; y < h; y++ {
		row := buf[y*stride:]
		for x := 0; x < w; x++ {
			i := x * 4
			row[i] = byte(x * 255 / w)
			row[i+1] = byte(y * 255 / h)
			row[i+2] = 220
			row[i+3] = 255
		}
	}
	fmt.Println("paint: writing")
	n, err := f.WriteAt(buf, 0)
	fmt.Println("paint: wrote", n, err)
	if err := ioctl(f.Fd(), 0x4606, unsafe.Pointer(&v)); err != nil {
		fmt.Println("paint: pan", err)
	} else {
		fmt.Println("paint: pan ok")
	}
	select {}
}
`

func writeDyn(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/ukiapp\n\ngo 1.27.0\n"), 0o644))
	src := "package main\n\nimport \"fmt\"\n\n" +
		"//go:cgo_import_dynamic _ _ \"libc.so.6\"\n" +
		"//go:cgo_import_dynamic _ _ \"libdl.so.2\"\n" +
		"//go:cgo_import_dynamic _ _ \"libpthread.so.0\"\n\n" +
		"func main() { fmt.Println(\"dyn app\") }\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644))
	return dir
}

func TestUKIBoots(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	out := filepath.Join(t.TempDir(), "app.efi")
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     writeTiny(t),
		Out:     out,
		AppID:   "br.tec.lew.ukiapp",
		Version: "0.1.0",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	serial := filepath.Join(t.TempDir(), "serial.log")
	bootCtx, bootCancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer bootCancel()
	err = boot(bootCtx, out, serial, "")
	raw, readErr := os.ReadFile(serial)
	require.NoError(t, readErr)
	if err != nil {
		t.Logf("qemu: %v", err)
	}
	require.Contains(t, string(raw), "uki app")
}

func TestISOBoots(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	out := filepath.Join(t.TempDir(), "app.iso")
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     writeTiny(t),
		Out:     out,
		AppID:   "br.tec.lew.ukiapp",
		Version: "0.1.0",
		GOOS:    "linux_iso",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	raw, err := os.ReadFile(out)
	require.NoError(t, err)
	require.Equal(t, "CD001", string(raw[0x8001:0x8006]))
	serial := filepath.Join(t.TempDir(), "serial.log")
	bootCtx, bootCancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer bootCancel()
	err = boot(bootCtx, out, serial, "")
	log, readErr := os.ReadFile(serial)
	require.NoError(t, readErr)
	if err != nil {
		t.Logf("qemu: %v", err)
	}
	require.Contains(t, string(log), "uki app")
}

func TestUTMDisplayAndKeyboard(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	out := filepath.Join(t.TempDir(), "app.iso")
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     writeTiny(t),
		Out:     out,
		AppID:   "br.tec.lew.ukiapp",
		Version: "0.1.0",
		GOOS:    "linux_iso",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	serial := filepath.Join(t.TempDir(), "serial.log")
	bootCtx, bootCancel := context.WithTimeout(t.Context(), 60*time.Second)
	defer bootCancel()
	// UTM's default window is ramfb or virtio, and the keys go to a USB HID device.
	err = bootArgs(bootCtx, out, serial, "", []string{
		"-vga", "none",
		"-device", "ramfb",
		"-device", "virtio-gpu-pci",
		"-device", "qemu-xhci,id=xhci",
		"-device", "usb-kbd,bus=xhci.0",
		"-device", "usb-ehci,id=ehci",
		"-device", "usb-kbd,bus=ehci.0",
	})
	log, readErr := os.ReadFile(serial)
	require.NoError(t, readErr)
	if err != nil {
		t.Logf("qemu: %v\n%s", err, log)
	}
	text := strings.ReplaceAll(string(log), "\r\n", "\n")
	require.Contains(t, text, "uki: fb0 ready")
	require.Contains(t, text, "Keyboard")
	require.NotContains(t, text, "uki: insmod")
}

func TestRamfbPaints(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	out := filepath.Join(t.TempDir(), "paint.iso")
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     writePaint(t),
		Out:     out,
		AppID:   "br.tec.lew.ukiapp",
		Version: "0.1.0",
		GOOS:    "linux_iso",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	serial := filepath.Join(t.TempDir(), "serial.log")
	shot := filepath.Join(t.TempDir(), "shot.ppm")
	bootCtx, bootCancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer bootCancel()
	err = bootArgs(bootCtx, out, serial, shot, []string{
		"-vga", "none",
		"-device", "ramfb",
	})
	log, readErr := os.ReadFile(serial)
	require.NoError(t, readErr)
	if err != nil {
		t.Logf("qemu: %v\n%s", err, log)
		if raw, readShot := os.ReadFile(shot); readShot == nil {
			head := raw
			if len(head) > 80 {
				head = head[:80]
			}
			t.Logf("shot %d bytes head %q", len(raw), head)
		}
	}
	require.NoError(t, err)
	require.Contains(t, string(log), "paint: mmap")
	require.Contains(t, string(log), "paint: wrote")
	require.Contains(t, string(log), "paint: pan ok")
}

func TestDVDDrawsOnRamfb(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	out := filepath.Join(t.TempDir(), "dvd.iso")
	ctx, cancel := context.WithTimeout(t.Context(), 6*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     filepath.Join(repoRoot(t), "examples/app/dvd"),
		Out:     out,
		AppID:   "br.tec.lew.eletrocromo.dvd",
		Version: "0.1.0",
		GOOS:    "linux_iso",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	serial := filepath.Join(t.TempDir(), "serial.log")
	shot := filepath.Join(t.TempDir(), "shot.ppm")
	bootCtx, bootCancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer bootCancel()
	// UTM's default window is ramfb (simpledrm). The USB devices are its keyboard.
	err = bootArgs(bootCtx, out, serial, shot, []string{
		"-vga", "none",
		"-device", "ramfb",
		"-device", "qemu-xhci,id=xhci",
		"-device", "usb-tablet,bus=xhci.0",
		"-device", "usb-kbd,bus=xhci.0",
	})
	log, readErr := os.ReadFile(serial)
	require.NoError(t, readErr)
	if err != nil {
		t.Logf("qemu: %v\n%s", err, log)
		if raw, readShot := os.ReadFile(shot); readShot == nil {
			head := raw
			if len(head) > 64 {
				head = head[:64]
			}
			t.Logf("shot %d bytes head %q", len(raw), head)
		}
	}
	require.NoError(t, err)
	require.Contains(t, string(log), "kms:")
}

func TestDynamicISOBoots(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	out := filepath.Join(t.TempDir(), "dyn.iso")
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     writeDyn(t),
		Out:     out,
		AppID:   "br.tec.lew.ukiapp",
		Version: "0.1.0",
		GOOS:    "linux_iso",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	serial := filepath.Join(t.TempDir(), "serial.log")
	bootCtx, bootCancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer bootCancel()
	err = boot(bootCtx, out, serial, "")
	log, readErr := os.ReadFile(serial)
	require.NoError(t, readErr)
	if err != nil {
		t.Logf("qemu: %v\n%s", err, log)
	}
	require.Contains(t, string(log), "dyn app")
}

func TestChessDraws(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	root := repoRoot(t)
	out := filepath.Join(t.TempDir(), "chess.efi")
	ctx, cancel := context.WithTimeout(t.Context(), 12*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     filepath.Join(root, "examples/app/chess"),
		Out:     out,
		AppID:   "br.tec.lew.eletrocromo.chess",
		Version: "0.1.0",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	shot := filepath.Join(t.TempDir(), "shot.ppm")
	serial := filepath.Join(t.TempDir(), "serial.log")
	err = boot(ctx, out, serial, shot)
	if err != nil {
		tail, _ := os.ReadFile(serial)
		t.Logf("serial:\n%s", tail)
	}
	require.NoError(t, err)
	require.True(t, frameDrawn(shot))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../.."))
	_, err := os.Stat(filepath.Join(root, "go.mod"))
	require.NoError(t, err)
	return root
}

func TestUnikernelExampleISOWaits(t *testing.T) {
	if os.Getenv("QEMU_SYSTEM") == "" || os.Getenv("OVMF_CODE") == "" || os.Getenv("OVMF_VARS") == "" {
		test.Skip(t, "QEMU_SYSTEM or OVMF is unset")
	}
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "this boot uses qemu-system-x86_64")
	}
	out := filepath.Join(t.TempDir(), "unikernel.iso")
	ctx, cancel := context.WithTimeout(t.Context(), 4*time.Minute)
	defer cancel()
	err := Build(ctx, Options{
		Dir:     filepath.Join(repoRoot(t), "examples/unikernel"),
		Out:     out,
		AppID:   "br.tec.lew.eletrocromo.unikernel",
		Version: "0.1.0",
		GOOS:    "linux_iso",
		GOARCH:  "amd64",
	})
	require.NoError(t, err)
	serial := filepath.Join(t.TempDir(), "serial.log")
	bootCtx, bootCancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer bootCancel()
	err = boot(bootCtx, out, serial, "")
	raw, readErr := os.ReadFile(serial)
	require.NoError(t, readErr)
	require.ErrorIs(t, err, context.DeadlineExceeded)
	text := strings.ReplaceAll(string(raw), "\r\n", "\n")
	require.Contains(t, text, "hello from the unikernel\n")
	require.Contains(t, text, "press enter to confirm\n")
	require.NotContains(t, text, "Kernel panic")
}

func TestDynamicLibsWithoutLdd(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/dyn\n\ngo 1.27.0\n"), 0o644))
	src := "package main\n\n" +
		"//go:cgo_import_dynamic _ _ \"libc.so.6\"\n" +
		"//go:cgo_import_dynamic _ _ \"libdl.so.2\"\n" +
		"//go:cgo_import_dynamic _ _ \"libpthread.so.0\"\n\n" +
		"func main() {}\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(src), 0o644))
	bin := filepath.Join(dir, "app")
	goBin, err := exec.LookPath("go")
	require.NoError(t, err)
	cmd := exec.Command(goBin, "build", "-o", bin, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
	f, err := elf.Open(bin)
	require.NoError(t, err)
	var interp bool
	for _, prog := range f.Progs {
		if prog.Type == elf.PT_INTERP {
			interp = true
		}
	}
	f.Close()
	require.True(t, interp)

	staticDir := writeTiny(t)
	staticBin := filepath.Join(t.TempDir(), "static")
	cmd = exec.Command(goBin, "build", "-o", staticBin, ".")
	cmd.Dir = staticDir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=linux", "GOARCH=amd64")
	out, err = cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	t.Setenv("PATH", t.TempDir())
	assertDynamicSet(t, bin)

	files, err := dynamicLibs(t.Context(), "amd64", staticBin)
	require.NoError(t, err)
	require.Empty(t, files)
}

func assertDynamicSet(t *testing.T, bin string) {
	t.Helper()
	files, err := dynamicLibs(t.Context(), "amd64", bin)
	require.NoError(t, err)
	saw := map[string][]byte{}
	for _, file := range files {
		require.True(t, isELF(file.data), file.name)
		saw[path.Base(file.name)] = file.data
	}
	for _, name := range []string{
		"libc.so.6", "libdl.so.2", "libpthread.so.0",
		"ld-linux-x86-64.so.2", "ld-musl-x86_64.so.1", "libgcompat.so.0",
	} {
		_, ok := saw[name]
		require.True(t, ok, "%s missing", name)
	}
	needed, err := elfNeeded(saw["libc.so.6"])
	require.NoError(t, err)
	require.Contains(t, needed, "libc.musl-x86_64.so.1")
}

func TestAlpineLibs(t *testing.T) {
	for _, goarch := range []string{"amd64", "arm64"} {
		t.Run(goarch, func(t *testing.T) {
			files, err := fetchAlpineLibs(t.Context(), goarch)
			require.NoError(t, err)
			var loader, musl string
			switch goarch {
			case "amd64":
				loader = "lib64/ld-linux-x86-64.so.2"
				musl = "lib/ld-musl-x86_64.so.1"
			default:
				loader = "lib/ld-linux-aarch64.so.1"
				musl = "lib/ld-musl-aarch64.so.1"
			}
			require.True(t, isELF(files[loader]), loader)
			require.True(t, isELF(files[musl]), musl)
			require.Equal(t, files["lib/libgcompat.so.0"], files["lib/libc.so.6"])
			require.Equal(t, files["lib/libgcompat.so.0"], files["lib/libdl.so.2"])
			require.Equal(t, files["lib/libgcompat.so.0"], files["lib/libpthread.so.0"])
			hits := sysrootHits(files, "/"+loader)
			require.Len(t, hits, 1)
			require.Equal(t, loader, hits[0].guest)
		})
	}
}

func TestFrameDrawnRejectsTextPage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.ppm")
	var b strings.Builder
	b.WriteString("P6\n720 400\n255\n")
	pix := make([]byte, 720*400*3)
	for i := 0; i < len(pix); i += 3 {
		pix[i] = 255
	}
	b.Write(pix)
	require.NoError(t, os.WriteFile(path, []byte(b.String()), 0o644))
	require.False(t, frameDrawn(path))
}
