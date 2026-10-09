package linux

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/release"
)

func TestBuildAppDirectory(t *testing.T) {
	mainDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "Demo.app")
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
	bin := filepath.Join(out, "Demo")
	info, err := os.Stat(bin)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatal("binary is not executable")
	}
	run, err := os.Stat(filepath.Join(out, "AppRun"))
	if err != nil {
		t.Fatal(err)
	}
	if run.Mode()&0o111 == 0 {
		t.Fatal("AppRun is not executable")
	}
	marker, err := os.ReadFile(filepath.Join(out, release.MarkerFile))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(marker)) != "br.tec.lew.demo" {
		t.Fatalf("marker %q", marker)
	}
	desktopPath := filepath.Join(out, "Demo.desktop")
	desktopInfo, err := os.Stat(desktopPath)
	if err != nil {
		t.Fatal(err)
	}
	if desktopInfo.Mode()&0o111 == 0 {
		t.Fatal("desktop file is not executable")
	}
	desktop, err := os.ReadFile(desktopPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(desktop)
	if !strings.Contains(text, "StartupWMClass=br.tec.lew.demo") {
		t.Fatalf("desktop %s", text)
	}
	if !strings.Contains(text, "Exec=\""+bin+"\"") {
		t.Fatalf("exec path %s", text)
	}
	icon := filepath.Join(out, "icons", "hicolor", "256x256", "apps", "br.tec.lew.demo.png")
	if _, err := os.Stat(icon); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "icon.png")); err != nil {
		t.Fatal(err)
	}
}
