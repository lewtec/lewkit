package apk

import (
	"debug/elf"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/lewtec/lewkit/x/build/gocmd"
	"github.com/lewtec/lewkit/x/build/version"
	"github.com/lewtec/lewkit/x/test"
	gotool "github.com/lewtec/lewkit/x/toolchain/go"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	gotool.TestExit(m)
}

func TestLewkitDirFindsModule(t *testing.T) {
	dir := lewkitDir()
	require.NotEmpty(t, dir)
	_, err := os.Stat(filepath.Join(dir, "x/entry/android/_cgo_android_export.go"))
	require.NoError(t, err)
}

func TestEntryExportFileUsesAndroidGOOS(t *testing.T) {
	t.Setenv("GOOS", runtime.GOOS)
	path, err := entryExportFile(t.Context())
	require.NoError(t, err)
	info, err := os.Stat(path)
	require.NoError(t, err)
	require.False(t, info.IsDir())
	require.Equal(t, "_cgo_android_export.go", filepath.Base(path))
}

func TestJoinToolexecQuotesSpaces(t *testing.T) {
	got, err := joinToolexec("/tmp/android toolexec", "/tmp/x/_cgo_android_export.go")
	require.NoError(t, err)
	require.Equal(t, "'/tmp/android toolexec' /tmp/x/_cgo_android_export.go", got)
	_, err = joinToolexec("", "x")
	require.Error(t, err)
}

func TestNocgoArm64ExportsJNIOnLoad(t *testing.T) {
	mini, err := filepath.Abs("testdata/nocgo")
	require.NoError(t, err)
	plain := filepath.Join(t.TempDir(), "plain")
	cmd := exec.Command("go", "build", "-trimpath", "-o", plain, ".")
	cmd.Dir = mini
	cmd.Env = append(os.Environ(), "GOOS=android", "GOARCH=arm64", "CGO_ENABLED=0")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "plain build:\n%s", out)
	ok, err := dynDefined(plain, "JNI_OnLoad")
	require.NoError(t, err)
	require.False(t, ok, "build without the wrapper exported JNI_OnLoad")

	libs, err := BuildGoLibs(t.Context(), t.TempDir(), mini, []string{"arm64-v8a"}, version.Info{}, "br.tec.lew.nocgo", false, io.Discard)
	require.NoError(t, err)
	require.Len(t, libs, 1)
	ok, err = dynDefined(libs[0], "JNI_OnLoad")
	require.NoError(t, err)
	require.True(t, ok, "cgo-free arm64 library has no JNI_OnLoad in .dynsym")
}

func TestNocgoArm64ExportsJNIOnLoadLewkit(t *testing.T) {
	if runtime.GOARCH != "amd64" {
		test.Skip(t, "embedded compiler runs on amd64")
	}
	mini, err := filepath.Abs("testdata/nocgo")
	require.NoError(t, err)
	ctx := gocmd.WithCompiler(t.Context(), gocmd.CompilerLewkit)
	libs, err := BuildGoLibs(ctx, t.TempDir(), mini, []string{"arm64-v8a"}, version.Info{}, "br.tec.lew.nocgo", false, io.Discard)
	require.NoError(t, err)
	require.Len(t, libs, 1)
	for _, name := range []string{"JNI_OnLoad", "Java_lewkit_Hook_call", "Java_lewkit_GoProxy_nativeInvoke"} {
		ok, err := dynDefined(libs[0], name)
		require.NoError(t, err)
		require.True(t, ok, "%s missing from .dynsym", name)
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
