package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrailerRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Demo.AppImage")
	if err := os.WriteFile(path, []byte("\x7fELF-demo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenTrailer(path); err == nil {
		t.Fatal("plain executable opened as a trailer")
	}
	err := AppendTrailer(path, map[string][]byte{
		MarkerFile: []byte("br.tec.lew.demo\n"),
		"icon.png": []byte("png"),
	})
	if err != nil {
		t.Fatal(err)
	}
	tr, err := OpenTrailer(path)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	id, err := tr.Bytes(MarkerFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(id) != "br.tec.lew.demo\n" {
		t.Fatalf("id %q", id)
	}
	icon, err := tr.Bytes("icon.png")
	if err != nil || string(icon) != "png" {
		t.Fatalf("icon %q %v", icon, err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw[:4]) != "\x7fELF" {
		t.Fatalf("prefix %q", raw[:4])
	}
}
