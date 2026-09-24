// Package pulse binds libpulse and libpulse-simple without cgo.
package pulse

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"unsafe"

	"github.com/lewtec/lewkit/x/ffi/native"
)

var (
	errUnavailable = errors.New("pulse unavailable")
	errClosed      = errors.New("pulse stream closed")
	errMainloop    = errors.New("pulse mainloop")
	errContext     = errors.New("pulse context")
	errSinkList    = errors.New("pulse sink list")
	errTimeout     = errors.New("pulse timeout")
	errNoPath      = errors.New("no library path")
)

// Sample matches pa_sample_format_t values used for playback.
type Sample int32

const (
	// SampleS16LE is PA_SAMPLE_S16LE.
	SampleS16LE Sample = 3
	// SampleF32LE is PA_SAMPLE_FLOAT32LE.
	SampleF32LE Sample = 5
)

// Sink is one PulseAudio or PipeWire playback device.
type Sink struct {
	Name        string
	Description string
}

func openLib(soname string) (uintptr, error) {
	var last error
	for _, path := range libPaths(soname) {
		lib, err := native.Open(path, native.Lazy)
		if err == nil {
			return lib, nil
		}
		last = err
	}
	if last == nil {
		last = errNoPath
	}
	return 0, fmt.Errorf("%s: %w", soname, last)
}

func libPaths(soname string) []string {
	paths := []string{soname}
	if dir := os.Getenv("LEWKIT_LIB"); dir != "" {
		paths = append(paths, filepath.Join(dir, soname))
	}
	return append(paths, filepath.Join("/run/current-system/sw/lib", soname))
}

func bind(lib uintptr, name string, fnptr any) error {
	if _, err := native.Symbol(lib, name); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	native.Func(lib, name, fnptr)
	return nil
}

func cString(s string) []byte {
	if s == "" {
		return nil
	}
	out := make([]byte, len(s)+1)
	copy(out, s)
	return out
}

func ptr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

func goString(p uintptr) string {
	if p == 0 {
		return ""
	}
	n := 0
	for *(*byte)(unsafe.Pointer(p + uintptr(n))) != 0 {
		n++
		if n > 1<<20 {
			break
		}
	}
	if n == 0 {
		return ""
	}
	return string(unsafe.Slice((*byte)(unsafe.Pointer(p)), n))
}
