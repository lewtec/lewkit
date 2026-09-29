package build

import (
	"debug/pe"
	"errors"
	"image"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lewtec/lewkit/x/build/icons"
	"github.com/stretchr/testify/require"
	"github.com/tc-hib/winres"
)

func TestEmbedWindowsIcon(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tiny\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	exe := filepath.Join(t.TempDir(), "tiny.exe")
	cmd := exec.Command("go", "build", "-trimpath", "-o", exe, ".")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0", "GOOS=windows", "GOARCH=amd64", "GOTOOLCHAIN=local")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))

	const manifest = `<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0"></assembly>`
	require.NoError(t, rewriteEXE(exe, func(rs *winres.ResourceSet) error {
		rs.Set(winres.RT_MANIFEST, winres.ID(1), winres.LCIDDefault, []byte(manifest))
		return nil
	}))
	before := windowsManifest(t, exe)
	require.Equal(t, []byte(manifest), before)
	ico := filepath.Join(t.TempDir(), "icon.ico")
	require.NoError(t, icons.WriteICO(ico, image.NewNRGBA(image.Rect(0, 0, 32, 32)), []int{16, 32}))
	require.NoError(t, embedWindowsIcon(exe, ico))

	file, err := os.Open(exe)
	require.NoError(t, err)
	defer file.Close()
	rs, err := winres.LoadFromEXE(file)
	require.NoError(t, err)
	icon, err := rs.GetIcon(winres.ID(1))
	require.NoError(t, err)
	require.NotNil(t, icon)
	require.Equal(t, before, manifestBytes(rs))
}

func TestWindowsAppIsGUIWithIcon(t *testing.T) {
	t.Setenv("GOTOOLCHAIN", "local")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tiny\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "eletrocromo.json"), []byte("{\n  \"schema_version\": 1,\n  \"package_id\": \"br.tec.lew.tiny\",\n  \"app_name\": \"Tiny\",\n  \"version_name\": \"0.1.0\",\n  \"go_main\": \".\"\n}\n"), 0o644))
	cwd := t.TempDir()
	t.Chdir(cwd)
	rel := filepath.Join("dist", "Tiny.exe")
	got, err := Host{
		Spec:   Spec{Dir: dir},
		Out:    rel,
		GOARCH: "amd64",
	}.Windows(t.Context())
	require.NoError(t, err)
	exe, err := filepath.Abs(rel)
	require.NoError(t, err)
	require.Equal(t, exe, got)
	_, err = os.Stat(filepath.Join(dir, rel))
	require.True(t, os.IsNotExist(err))
	require.Equal(t, uint16(pe.IMAGE_SUBSYSTEM_WINDOWS_GUI), peSubsystem(t, exe))

	file, err := os.Open(exe)
	require.NoError(t, err)
	defer file.Close()
	rs, err := winres.LoadFromEXE(file)
	require.NoError(t, err)
	icon, err := rs.GetIcon(winres.ID(1))
	require.NoError(t, err)
	require.NotNil(t, icon)
}

func TestWindowsArchiveStaysConsole(t *testing.T) {
	t.Setenv("GOTOOLCHAIN", "local")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/tiny\n\ngo 1.27.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644))
	exe := filepath.Join(t.TempDir(), "tiny.exe")
	require.NoError(t, goBinary{
		dir:     dir,
		appID:   "br.tec.lew.tiny",
		version: "0.1.0",
		goos:    "windows",
		goarch:  "amd64",
		dest:    exe,
	}.compile(t.Context()))
	require.Equal(t, uint16(pe.IMAGE_SUBSYSTEM_WINDOWS_CUI), peSubsystem(t, exe))
}

func peSubsystem(t *testing.T, path string) uint16 {
	t.Helper()
	file, err := pe.Open(path)
	require.NoError(t, err)
	defer file.Close()
	switch header := file.OptionalHeader.(type) {
	case *pe.OptionalHeader64:
		return header.Subsystem
	case *pe.OptionalHeader32:
		return header.Subsystem
	default:
		t.Fatalf("optional header %T", file.OptionalHeader)
		return 0
	}
}

func TestWriteLinuxDesktop(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "Basic")
	require.NoError(t, os.WriteFile(bin, []byte("bin"), 0o755))
	root := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(root, "linux"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, "linux", "icon-256.png"), []byte("png256"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "linux", "icon-512.png"), []byte("png512"), 0o644))
	require.NoError(t, writeLinuxDesktop(bin, "Basic\nApp", root))

	got, err := os.ReadFile(filepath.Join(dir, "Basic.png"))
	require.NoError(t, err)
	require.Equal(t, "png512", string(got))
	absBin, err := filepath.Abs(bin)
	require.NoError(t, err)
	absPNG, err := filepath.Abs(filepath.Join(dir, "Basic.png"))
	require.NoError(t, err)
	desk, err := os.ReadFile(filepath.Join(dir, "Basic.desktop"))
	require.NoError(t, err)
	require.Equal(t, "[Desktop Entry]\nType=Application\nName=Basic App\nExec="+absBin+"\nIcon="+absPNG+"\nTerminal=false\n", string(desk))
}

func TestDesktopExecQuotesSpaces(t *testing.T) {
	require.Equal(t, "/tmp/Basic", desktopExec("/tmp/Basic"))
	require.Equal(t, `"/tmp/My App"`, desktopExec("/tmp/My App"))
}

func windowsManifest(t *testing.T, path string) []byte {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	rs, err := winres.LoadFromEXE(file)
	if errors.Is(err, winres.ErrNoResources) {
		return nil
	}
	require.NoError(t, err)
	return manifestBytes(rs)
}

func manifestBytes(rs *winres.ResourceSet) []byte {
	var body []byte
	rs.WalkType(winres.RT_MANIFEST, func(_ winres.Identifier, _ uint16, data []byte) bool {
		body = append([]byte(nil), data...)
		return false
	})
	return body
}
