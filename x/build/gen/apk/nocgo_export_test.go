package apk

import (
	"debug/elf"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/version"
)

func TestJoinToolexecQuotesSpaces(t *testing.T) {
	got, err := joinToolexec("/tmp/android toolexec", "/tmp/x/_cgo_android_export.go")
	if err != nil {
		t.Fatal(err)
	}
	want := "'/tmp/android toolexec' /tmp/x/_cgo_android_export.go"
	if got != want {
		t.Fatalf("joinToolexec() = %q, want %q", got, want)
	}
	if _, err := joinToolexec("", "x"); err == nil {
		t.Fatal("empty bin returned nil")
	}
}

func TestNocgoArm64ExportsJNIOnLoad(t *testing.T) {
	mini, err := filepath.Abs("testdata/nocgo")
	if err != nil {
		t.Fatal(err)
	}
	poison := filepath.Join(t.TempDir(), "plain")
	cmd := exec.Command("go", "build", "-trimpath", "-o", poison, ".")
	cmd.Dir = mini
	cmd.Env = append(os.Environ(), "GOOS=android", "GOARCH=arm64", "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("plain build: %v\n%s", err, out)
	}
	ok, err := dynDefined(poison, "JNI_OnLoad")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("build without the wrapper exported JNI_OnLoad")
	}

	libs, err := BuildGoLibs(t.Context(), t.TempDir(), mini, []string{"arm64-v8a"}, version.Info{}, "br.tec.lew.nocgo", false, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	if len(libs) != 1 {
		t.Fatalf("libs = %#v", libs)
	}
	ok, err = dynDefined(libs[0], "JNI_OnLoad")
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("cgo-free arm64 library has no JNI_OnLoad in .dynsym")
	}
}

func dynDefined(path, name string) (bool, error) {
	f, err := elf.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()
	syms, err := f.DynamicSymbols()
	if err != nil {
		return false, err
	}
	for _, sym := range syms {
		if sym.Name == name && sym.Section != elf.SHN_UNDEF {
			return true, nil
		}
	}
	return false, nil
}
