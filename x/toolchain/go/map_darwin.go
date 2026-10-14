//go:build darwin

package gotool

import (
	"errors"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"golang.org/x/sys/unix"
)

var (
	ErrJitExecutableMappingIsUnavailable    = errors.New("jit: executable mapping is unavailable")
	ErrJitPthreadJitWriteProtectNpIsMissing = errors.New("jit: pthread_jit_write_protect_np is missing")
)

// darwinJIT is set by mapNear. MAP_JIT pages flip between writable and
// executable with pthread_jit_write_protect_np. A plain anonymous
// mapping uses mprotect instead.
var darwinJIT bool

func prepareWrite(mem []byte) error {
	if len(mem) == 0 || !darwinJIT {
		return nil
	}
	return setJITWritable(true)
}

func protectRX(mem []byte) error {
	if len(mem) == 0 {
		return nil
	}
	if darwinJIT {
		return setJITWritable(false)
	}
	return unix.Mprotect(mem, unix.PROT_READ|unix.PROT_EXEC)
}

func mapNear(n int, hint uintptr) ([]byte, error) {
	page := uintptr(unix.Getpagesize())
	size := (uintptr(n) + page - 1) &^ (page - 1)
	try := func(addr uintptr, prot, flags int) ([]byte, bool) {
		if addr != 0 && (addr < page || addr+size < addr) {
			return nil, false
		}
		p, ok := mmapAt(addr, size, prot, flags)
		if !ok {
			return nil, false
		}
		return unsafe.Slice((*byte)(unsafe.Pointer(p)), size), true
	}
	jitProt := unix.PROT_READ | unix.PROT_EXEC
	jitFlags := unix.MAP_PRIVATE | unix.MAP_ANON | unix.MAP_JIT
	limit := uintptr(1 << 31)
	if runtime.GOARCH == "arm64" {
		limit = 96 << 20
	}
	const step = 2 << 20
	base := hint &^ (page - 1)
	// MAP_JIT pages stay read/execute until this thread calls
	// pthread_jit_write_protect_np(0). Without that symbol, an
	// anonymous read/write mapping is the one mprotect can flip.
	if jitWriteOK() {
		for dist := uintptr(0); dist < limit; dist += step {
			cands := []uintptr{base + dist}
			if dist > 0 && base > dist {
				cands = append(cands, base-dist)
			}
			for _, addr := range cands {
				if b, ok := try(addr, jitProt, jitFlags); ok {
					darwinJIT = true
					return b, nil
				}
			}
		}
		if b, ok := try(0, jitProt, jitFlags); ok {
			darwinJIT = true
			return b, nil
		}
	}
	darwinJIT = false
	if b, ok := try(0, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON); ok {
		return b, nil
	}
	return nil, ErrJitExecutableMappingIsUnavailable
}

func mmapAt(addr, size uintptr, prot, flags int) (uintptr, bool) {
	p, _, e := unix.Syscall6(unix.SYS_MMAP, addr, size, uintptr(prot), uintptr(flags), ^uintptr(0), 0)
	if e != 0 || p == ^uintptr(0) {
		return 0, false
	}
	if addr != 0 && p != addr {
		unix.Syscall(unix.SYS_MUNMAP, p, size, 0)
		return 0, false
	}
	return p, true
}

var (
	jitOnce      sync.Once
	jitWrite     func(enabled int)
	jitSupported func() int
	jitErr       error
)

func jitWriteOK() bool {
	write, supported, err := loadJITWrite()
	if err != nil || write == nil {
		return false
	}
	if supported != nil && supported() == 0 {
		return false
	}
	return true
}

func loadJITWrite() (func(int), func() int, error) {
	jitOnce.Do(func() {
		lib, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
		if err != nil {
			jitErr = err
			return
		}
		sym, err := purego.Dlsym(lib, "pthread_jit_write_protect_np")
		if err != nil {
			jitErr = err
			return
		}
		purego.RegisterFunc(&jitWrite, sym)
		if supported, err := purego.Dlsym(lib, "pthread_jit_write_protect_supported_np"); err == nil {
			purego.RegisterFunc(&jitSupported, supported)
		}
	})
	if jitErr != nil {
		return nil, nil, jitErr
	}
	return jitWrite, jitSupported, nil
}

func setJITWritable(writable bool) error {
	write, _, err := loadJITWrite()
	if err != nil {
		return err
	}
	if write == nil {
		return ErrJitPthreadJitWriteProtectNpIsMissing
	}
	// 0 makes MAP_JIT pages writable on this thread. 1 makes them executable.
	enabled := 1
	if writable {
		enabled = 0
	}
	write(enabled)
	return nil
}
