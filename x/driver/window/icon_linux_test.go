//go:build linux

package window

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestReadIconPNG(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.SetNRGBA(0, 0, color.NRGBA{R: 10, G: 20, B: 30, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "icon.png")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readIcon(path)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, a := got.At(0, 0).RGBA()
	if r>>8 != 10 || g>>8 != 20 || b>>8 != 30 || a>>8 != 255 {
		t.Fatalf("pixel %d %d %d %d", r, g, b, a)
	}
}

func TestApplyXIconSkipsWithoutWindow(t *testing.T) {
	if err := ApplyXIcon(0, image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
}
