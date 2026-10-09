package linux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/release"
)

func TestBuildNilContextPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("nil context returned")
		}
	}()
	_, _ = Build(nil, BuildOptions{})
}

func TestBuildAppImage(t *testing.T) {
	mainDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "Demo.AppImage")
	result, err := Build(t.Context(), BuildOptions{
		Config: Config{
			PackageID:   "br.tec.lew.demo",
			AppName:     "Demo",
			VersionName: "1.2.3",
			VersionCode: 5,
			GoMain:      mainDir,
		},
		BaseDir: mainDir,
		WorkDir: t.TempDir(),
		OutApp:  out,
		GOARCH:  "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.AppPath != out {
		t.Fatalf("app %s", result.AppPath)
	}
	info, err := os.Stat(out)
	if err != nil {
		t.Fatal(err)
	}
	if info.IsDir() {
		t.Fatal("app image is a directory")
	}
	if info.Mode()&0o111 == 0 {
		t.Fatal("app image is not executable")
	}
	raw, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw[:4]) != "\x7fELF" {
		t.Fatalf("prefix %q", raw[:4])
	}
	tr, err := release.OpenTrailer(out)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	marker, err := tr.Bytes(release.MarkerFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(marker)) != "br.tec.lew.demo" {
		t.Fatalf("marker %q", marker)
	}
	desktop, err := tr.Bytes("Demo.desktop")
	if err != nil {
		t.Fatal(err)
	}
	text := string(desktop)
	if !strings.Contains(text, "StartupWMClass=br.tec.lew.demo") || !strings.Contains(text, "Exec=Demo") || !strings.Contains(text, "Icon=icon") {
		t.Fatalf("desktop %s", text)
	}
	icon, err := tr.Bytes("icon.png")
	if err != nil || len(icon) == 0 {
		t.Fatalf("icon %v", err)
	}
}
