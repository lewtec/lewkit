package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPeelExport(t *testing.T) {
	rest, export := peelExport([]string{
		"/tmp/x/_cgo_android_export.go",
		"/usr/lib/compile",
		"-V=full",
	})
	if export != "/tmp/x/_cgo_android_export.go" {
		t.Fatalf("export = %q", export)
	}
	if len(rest) != 2 || rest[0] != "/usr/lib/compile" || rest[1] != "-V=full" {
		t.Fatalf("rest = %#v", rest)
	}
	rest, export = peelExport([]string{"/usr/lib/compile", "-p", entryPkg})
	if export != "" || len(rest) != 3 {
		t.Fatalf("export %q rest %#v", export, rest)
	}
}

func TestVersionProbe(t *testing.T) {
	if !versionProbe([]string{"/usr/lib/compile", "-V=full"}) {
		t.Fatal("probe not recognized")
	}
	if versionProbe([]string{"/usr/lib/compile", "-o", "a.o", "-V=full"}) {
		t.Fatal("compile line treated as a version probe")
	}
}

func TestExportStampTracksBytes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "_cgo_android_export.go")
	if err := os.WriteFile(path, []byte("package entry\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	first, err := exportStamp(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("package entry\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := exportStamp(path)
	if err != nil {
		t.Fatal(err)
	}
	if first == second || first == "" {
		t.Fatalf("stamp did not change: %q %q", first, second)
	}
}
