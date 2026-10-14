package gotool

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"testing"
	"unsafe"

	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/objabi"
	"github.com/stretchr/testify/require"
)

func TestStepExitIgnoresANormalTest(t *testing.T) {
	t.Setenv(actionEnv, "")
	handled, code := stepExit()
	require.False(t, handled)
	require.Equal(t, 0, code)
}

func TestHostSymResolvesMorestack(t *testing.T) {
	pc, err := hostFunc("runtime.morestack_noctxt", 0)
	require.NoError(t, err)
	if pc == 0 {
		require.FailNow(t, fmt.Sprint("morestack address is zero"))
	}
}

func TestTrampolineEncodes(t *testing.T) {
	var amd [16]byte
	emitAMD64Trampoline(amd[:], 0x1122334455667788)
	wantAMD := []byte{0x49, 0xBB, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x41, 0xFF, 0xE3}
	if string(amd[:len(wantAMD)]) != string(wantAMD) {
		require.FailNowf(t, "", "amd64 %x", amd[:len(wantAMD)])
	}
	var arm [16]byte
	emitARM64Trampoline(arm[:], 0x1122334455667788)
	if binary.LittleEndian.Uint32(arm[0:]) != 0x58000050 || binary.LittleEndian.Uint32(arm[4:]) != 0xD61F0200 {
		require.FailNowf(t, "", "arm64 insns %x", arm[:8])
	}
	if binary.LittleEndian.Uint64(arm[8:]) != 0x1122334455667788 {
		require.FailNowf(t, "", "arm64 target %x", arm[8:])
	}
}

func TestARM64RelocBits(t *testing.T) {
	call := []byte{0x00, 0x00, 0x00, 0x94}
	p := placed{addr: uintptr(unsafe.Pointer(&call[0])), size: len(call)}
	require.NoError(t, applyCallARM64(p, 0, int64(p.addr)+8, nil))
	if binary.LittleEndian.Uint32(call) != 0x94000002 {
		require.FailNowf(t, "", "bl %x", call)
	}

	addr := []byte{0x00, 0x00, 0x00, 0x90, 0x00, 0x00, 0x00, 0x91}
	p = placed{addr: uintptr(unsafe.Pointer(&addr[0])), size: len(addr)}
	target := (int64(p.addr) &^ 0xfff) + 0x234
	require.NoError(t, applyAddrARM64(p, 0, objabi.R_ADDRARM64, target))
	if binary.LittleEndian.Uint32(addr[0:]) != 0x90000000 || binary.LittleEndian.Uint32(addr[4:]) != 0x9108D000 {
		require.FailNowf(t, "", "adrp+add %x", addr)
	}
}

func TestCallChain(t *testing.T) {
	prog := compileMain(t, `
package main

//go:noinline
func Add(a, b int) int { return a + b }

func Value() string { return "jit-ok\n" }

func Sum() int { return Add(20, 22) }

func Recur(n int) int {
	if n == 0 {
		return 0
	}
	return Recur(n-1) + 1
}

func main() {}
`)
	value, err := Bind[func() string](prog, "Value")
	require.NoError(t, err)
	add, err := Bind[func(int, int) int](prog, "Add")
	require.NoError(t, err)
	sum, err := Bind[func() int](prog, "main.Sum")
	require.NoError(t, err)
	recur, err := Bind[func(int) int](prog, "Recur")
	require.NoError(t, err)
	if value() != "jit-ok\n" {
		require.FailNowf(t, "", "value %q", value())
	}
	if add(2, 3) != 5 {
		require.FailNowf(t, "", "add %d", add(2, 3))
	}
	if sum() != 42 {
		require.FailNowf(t, "", "sum %d", sum())
	}
	if recur(4000) != 4000 {
		require.FailNowf(t, "", "recur %d", recur(4000))
	}
	runtime.GC()
	if value() != "jit-ok\n" || add(1, 1) != 2 {
		require.FailNow(t, fmt.Sprint("call after gc"))
	}

	if _, err := CompileSource(t.Context(), "package main\nfunc Other() int { return 9 }\nfunc main() {}\n"); err == nil {
		require.FailNow(t, fmt.Sprint("second compile reused this process"))
	}
	if value() != "jit-ok\n" {
		require.FailNow(t, fmt.Sprint("first program after second compile"))
	}
}

func compileMain(t *testing.T, src string) *Prog {
	t.Helper()
	prog, err := CompileSource(t.Context(), src)
	require.NoError(t, err)
	return prog
}
