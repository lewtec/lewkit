package win

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildGUIExecutable(t *testing.T) {
	mainDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(mainDir, "go.mod"), []byte("module example.com/demo\n\ngo 1.27.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mainDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "Demo.exe")
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
		OutExe:  out,
		GOARCH:  "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExePath != out {
		t.Fatalf("exe %s", result.ExePath)
	}
	body, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	sub, err := peSubsystem(body)
	if err != nil {
		t.Fatal(err)
	}
	if sub != 2 {
		t.Fatalf("subsystem %d, want windows GUI", sub)
	}
	if !bytesContains(body, "permonitorv2,system") {
		t.Fatal("manifest missing permonitorv2")
	}
	if !bytesContains(body, "Demo") {
		t.Fatal("version info missing product name")
	}
	if !bytesContains(body, "IHDR") && !bytesContains(body, "PNG") {
		t.Fatal("icon payload missing")
	}
}

func peSubsystem(b []byte) (uint16, error) {
	if len(b) < 0x40 || b[0] != 'M' || b[1] != 'Z' {
		return 0, os.ErrInvalid
	}
	pe := int(binary.LittleEndian.Uint32(b[0x3C:]))
	if pe < 0 || pe+24+70 > len(b) || string(b[pe:pe+4]) != "PE\x00\x00" {
		return 0, os.ErrInvalid
	}
	magic := binary.LittleEndian.Uint16(b[pe+24:])
	if magic != 0x20B && magic != 0x10B {
		return 0, os.ErrInvalid
	}
	return binary.LittleEndian.Uint16(b[pe+24+68:]), nil
}

func bytesContains(b []byte, text string) bool {
	if strings.Contains(string(b), text) {
		return true
	}
	wide := make([]byte, len(text)*2)
	for i := 0; i < len(text); i++ {
		wide[i*2] = text[i]
	}
	return bytesIndex(b, wide)
}

func bytesIndex(b, sub []byte) bool {
	if len(sub) == 0 || len(sub) > len(b) {
		return false
	}
	for i := 0; i+len(sub) <= len(b); i++ {
		match := true
		for j := range sub {
			if b[i+j] != sub[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
