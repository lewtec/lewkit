//go:build linux

package entry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/release"
)

func TestPublishLauncherInstallsIcon(t *testing.T) {
	bundle := t.TempDir()
	exe := filepath.Join(bundle, "Demo")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, release.MarkerFile), []byte("br.tec.lew.demo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "icon.png"), []byte("png"), 0o644); err != nil {
		t.Fatal(err)
	}
	iconDir := filepath.Join(bundle, "icons", "hicolor", "256x256", "apps")
	if err := os.MkdirAll(iconDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(iconDir, "br.tec.lew.demo.png"), []byte("icon"), 0o644); err != nil {
		t.Fatal(err)
	}
	desktop := "[Desktop Entry]\nType=Application\nName=Demo\nExec=/old/Demo\nIcon=/old/icon.png\nStartupWMClass=br.tec.lew.demo\n"
	if err := os.WriteFile(filepath.Join(bundle, "Demo.desktop"), []byte(desktop), 0o644); err != nil {
		t.Fatal(err)
	}
	data := t.TempDir()
	if err := publishLauncher(exe, data); err != nil {
		t.Fatal(err)
	}
	installed, err := os.ReadFile(filepath.Join(data, "applications", "br.tec.lew.demo.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(installed)
	for _, line := range []string{
		`Exec="` + exe + `"`,
		"Icon=" + filepath.Join(bundle, "icon.png"),
		"StartupWMClass=br.tec.lew.demo",
		"Name=Demo",
	} {
		if !strings.Contains(text, line) {
			t.Fatalf("missing %q in\n%s", line, text)
		}
	}
	info, err := os.Stat(filepath.Join(bundle, "Demo.desktop"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&0o111 == 0 {
		t.Fatal("bundle desktop file is not executable")
	}
	copied := filepath.Join(data, "icons", "hicolor", "256x256", "apps", "br.tec.lew.demo.png")
	if _, err := os.Stat(copied); err != nil {
		t.Fatal(err)
	}
}
