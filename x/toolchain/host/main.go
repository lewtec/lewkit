// Command host runs the JIT, FFI, and cgo-replacement tests on this machine.
// Linux, Darwin, and Windows use the same list. An ELF-only test skips
// itself on the other two.
package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
)

var (
	ErrRun = errors.New("")
)

func main() {
	os.Setenv("CGO_ENABLED", "0")
	os.Setenv("GOTOOLCHAIN", "local")
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	suites := []struct {
		name string
		args []string
	}{
		{
			name: "jit and vendored link",
			args: []string{
				"test", "-count=1", "-timeout", "30m",
				"-run", `^(TestHostSymResolvesMorestack|TestTrampolineEncodes|TestARM64RelocBits|TestCompileSourceNil|TestCallChain|TestTargetImportCfg|TestPlanRejectsAssembly|TestBuildRejectsShape|TestBuildDependentPackages|TestVendoredExe|TestVendoredPIE|TestVendoredCShared)$`,
				"./x/toolchain/go",
			},
		},
		{
			name: "toolchain examples",
			args: []string{"test", "-count=1", "-timeout", "30m", "-run", `^(TestJIT|TestShare)$`, "./examples/toolchain"},
		},
		{name: "purego", args: []string{"test", "-count=1", "-timeout", "5m", "./x/toolchain/purego"}},
		{name: "c header", args: []string{"test", "-count=1", "-timeout", "5m", "./x/toolchain/c"}},
		{name: "load", args: []string{"test", "-count=1", "-timeout", "5m", "./x/toolchain/load"}},
		{name: "tamago", args: []string{"test", "-count=1", "-timeout", "5m", "./x/toolchain/tamago"}},
		{name: "ffi", args: []string{"test", "-count=1", "-timeout", "10m", "./x/ffi/native"}},
		{name: "ffi treesitter", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/native/treesitter"}},
		{name: "ffi vulkan", args: []string{"test", "-count=1", "-timeout", "10m", "./x/ffi/native/vulkan"}},
		{name: "ffi metal", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/native/metal"}},
		{name: "ffi d3d12", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/native/d3d12"}},
		{name: "ffi winmm", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/native/winmm"}},
		{name: "ffi coreaudio", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/native/coreaudio"}},
		{name: "ffi webview2", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/native/webview2"}},
		{name: "ffi opengl", args: []string{"test", "-count=1", "-timeout", "10m", "./x/ffi/native/opengl"}},
		{name: "ffi android", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/native/android"}},
		{name: "ffi jni", args: []string{"test", "-count=1", "-timeout", "5m", "./x/ffi/jni"}},
	}
	for _, suite := range suites {
		fmt.Printf("== %s\n", suite.name)
		cmd := exec.Command("go", suite.args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("%w%s: %w", ErrRun, suite.name, err)
		}
	}
	return nil
}
