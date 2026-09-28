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
	lib, err := native.OpenFirst(native.Lazy, libPaths(soname)...)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", soname, err)
	}
	return lib, nil
}

func libPaths(soname string) []string {
	paths := []string{soname}
	if dir := os.Getenv("LEWKIT_LIB"); dir != "" {
		paths = append(paths, filepath.Join(dir, soname))
	}
	return append(paths, filepath.Join("/run/current-system/sw/lib", soname))
}

func cString(s string) []byte {
	if s == "" {
		return nil
	}
	return native.CString(s)
}

func ptr(b []byte) uintptr {
	if len(b) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&b[0]))
}

func goString(p uintptr) string { return native.GoString(p) }
