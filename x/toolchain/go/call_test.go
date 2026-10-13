package gotool

import (
	"context"
	"encoding/binary"
	"errors"
	"runtime"
	"testing"
	"unsafe"

	"github.com/lewtec/lewkit/x/toolchain/go/src/cmd/objabi"
)

func TestHostSymResolvesMorestack(t *testing.T) {
	pc, err := hostFunc("runtime.morestack_noctxt", 0)
	if err != nil {
		t.Fatal(err)
	}
	if pc == 0 {
		t.Fatal("morestack address is zero")
	}
}

func TestTrampolineEncodes(t *testing.T) {
	var amd [16]byte
	emitAMD64Trampoline(amd[:], 0x1122334455667788)
	wantAMD := []byte{0x49, 0xBB, 0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0x41, 0xFF, 0xE3}
	if string(amd[:len(wantAMD)]) != string(wantAMD) {
		t.Fatalf("amd64 %x", amd[:len(wantAMD)])
	}
	var arm [16]byte
	emitARM64Trampoline(arm[:], 0x1122334455667788)
	if binary.LittleEndian.Uint32(arm[0:]) != 0x58000050 || binary.LittleEndian.Uint32(arm[4:]) != 0xD61F0200 {
		t.Fatalf("arm64 insns %x", arm[:8])
	}
	if binary.LittleEndian.Uint64(arm[8:]) != 0x1122334455667788 {
		t.Fatalf("arm64 target %x", arm[8:])
	}
}

func TestARM64RelocBits(t *testing.T) {
	call := []byte{0x00, 0x00, 0x00, 0x94}
	p := placed{addr: uintptr(unsafe.Pointer(&call[0])), size: len(call)}
	if err := applyCallARM64(p, 0, int64(p.addr)+8, nil); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(call) != 0x94000002 {
		t.Fatalf("bl %x", call)
	}

	addr := []byte{0x00, 0x00, 0x00, 0x90, 0x00, 0x00, 0x00, 0x91}
	p = placed{addr: uintptr(unsafe.Pointer(&addr[0])), size: len(addr)}
	target := (int64(p.addr) &^ 0xfff) + 0x234
	if err := applyAddrARM64(p, 0, objabi.R_ADDRARM64, target); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint32(addr[0:]) != 0x90000000 || binary.LittleEndian.Uint32(addr[4:]) != 0x9108D000 {
		t.Fatalf("adrp+add %x", addr)
	}
}

func TestCompileSourceNil(t *testing.T) {
	if _, err := CompileSource(nil, "package main\n"); !errors.Is(err, ErrNilContext) {
		t.Fatalf("err %v", err)
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
	if err != nil {
		t.Fatal(err)
	}
	add, err := Bind[func(int, int) int](prog, "Add")
	if err != nil {
		t.Fatal(err)
	}
	sum, err := Bind[func() int](prog, "main.Sum")
	if err != nil {
		t.Fatal(err)
	}
	recur, err := Bind[func(int) int](prog, "Recur")
	if err != nil {
		t.Fatal(err)
	}
	if value() != "jit-ok\n" {
		t.Fatalf("value %q", value())
	}
	if add(2, 3) != 5 {
		t.Fatalf("add %d", add(2, 3))
	}
	if sum() != 42 {
		t.Fatalf("sum %d", sum())
	}
	if recur(4000) != 4000 {
		t.Fatalf("recur %d", recur(4000))
	}
	runtime.GC()
	if value() != "jit-ok\n" || add(1, 1) != 2 {
		t.Fatal("call after gc")
	}

	if _, err := CompileSource(context.Background(), "package main\nfunc Other() int { return 9 }\nfunc main() {}\n"); err == nil {
		t.Fatal("second compile reused this process")
	}
	if value() != "jit-ok\n" {
		t.Fatal("first program after second compile")
	}
}

func compileMain(t *testing.T, src string) *Prog {
	t.Helper()
	prog, err := CompileSource(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	return prog
}
