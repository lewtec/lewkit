package icons

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestPadCenterSquare(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 50))
	for y := 0; y < 50; y++ {
		for x := 0; x < 100; x++ {
			img.Set(x, y, color.NRGBA{R: 255, A: 255})
		}
	}
	sq := PadCenter(img)
	if sq.Bounds().Dx() != 100 || sq.Bounds().Dy() != 100 {
		t.Fatalf("side got %v", sq.Bounds())
	}
}

func TestGenerateDefaultAndSkip(t *testing.T) {
	dir := t.TempDir()
	m1, err := Generate(Options{OutputDir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if m1.Source != "default" {
		t.Fatalf("source %q", m1.Source)
	}
	if !Complete(dir) {
		t.Fatal("expected complete tree")
	}
	// second call without force should skip (manifest still readable)
	m2, err := Generate(Options{OutputDir: dir, Force: false})
	if err != nil {
		t.Fatal(err)
	}
	if m2.GeneratedAt != m1.GeneratedAt {
		t.Fatalf("expected skip, times %s vs %s", m1.GeneratedAt, m2.GeneratedAt)
	}
	// force rebuild
	m3, err := Generate(Options{OutputDir: dir, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if m3.GeneratedAt == m1.GeneratedAt {
		// possible if same second; ensure files still ok
		if !Complete(dir) {
			t.Fatal("incomplete after force")
		}
	}
}

func TestGenerateFromPNG(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "in.png")
	img := image.NewNRGBA(image.Rect(0, 0, 64, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 64; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	f, err := os.Create(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "icons")
	m, err := Generate(Options{SourcePath: src, OutputDir: out, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if m.Source != src {
		t.Fatalf("source %q", m.Source)
	}
	if _, err := os.Stat(filepath.Join(out, "windows", "icon.ico")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(out, "macos", "icon.icns")); err != nil {
		t.Fatal(err)
	}
}

func TestApplyAndroidRes(t *testing.T) {
	iconsDir := t.TempDir()
	if _, err := Generate(Options{OutputDir: iconsDir, Force: true}); err != nil {
		t.Fatal(err)
	}
	res := filepath.Join(t.TempDir(), "res")
	if err := ApplyAndroidRes(iconsDir, res); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(res, "mipmap-mdpi", "ic_launcher.png")); err != nil {
		t.Fatal(err)
	}
}

func TestApplyMacOSICNS(t *testing.T) {
	iconsDir := t.TempDir()
	if _, err := Generate(Options{OutputDir: iconsDir, Force: true}); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(t.TempDir(), "Resources", "AppIcon.icns")
	if err := ApplyMacOSICNS(iconsDir, dest); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultLockupEmbed(t *testing.T) {
	if len(DefaultLockupPNG) < 100 {
		t.Fatal("default lockup embed empty")
	}
}

func TestDefaultAssetsHaveAlpha(t *testing.T) {
	t.Parallel()
	lockup, err := DecodeBytes(DefaultLockupPNG, "lockup.png")
	if err != nil {
		t.Fatal(err)
	}
	b := lockup.Bounds()
	_, _, _, a0 := lockup.At(b.Min.X, b.Min.Y).RGBA()
	if a0 != 0 {
		t.Fatalf("lockup corner should be transparent, a=%d", a0>>8)
	}
	mark, err := defaultMaster()
	if err != nil {
		t.Fatal(err)
	}
	mb := mark.Bounds()
	if mb.Dx() != 1024 || mb.Dy() != 1024 {
		t.Fatalf("mark size %v", mb)
	}
	_, _, _, ma := mark.At(mb.Min.X, mb.Min.Y).RGBA()
	if ma != 0 {
		t.Fatalf("mark corner should be transparent, a=%d", ma>>8)
	}
}

func TestExtractUpperMarkSplitsLockup(t *testing.T) {
	t.Parallel()
	img := image.NewNRGBA(image.Rect(0, 0, 40, 80))
	// top blob (mark)
	for y := 4; y < 20; y++ {
		for x := 8; x < 32; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	// bottom blob (wordmark)
	for y := 50; y < 60; y++ {
		for x := 4; x < 36; x++ {
			img.Set(x, y, color.NRGBA{R: 40, A: 255})
		}
	}
	got := ExtractUpperMark(img)
	if got.Bounds().Dx() != got.Bounds().Dy() {
		t.Fatalf("want square, got %v", got.Bounds())
	}
	// bottom of the square must not contain the wordmark band
	_, _, _, a := got.At(got.Bounds().Min.X+got.Bounds().Dx()/2, got.Bounds().Max.Y-1).RGBA()
	if a != 0 {
		t.Fatalf("bottom edge should be padding, a=%d", a>>8)
	}
	_, _, _, ac := got.At(got.Bounds().Dx()/2, got.Bounds().Dy()/2).RGBA()
	if ac < 0x8000 {
		t.Fatalf("center should keep the mark, a=%d", ac>>8)
	}
}

func TestContentBoundsAndTrim(t *testing.T) {
	t.Parallel()
	img := image.NewNRGBA(image.Rect(0, 0, 20, 20))
	for y := 5; y < 9; y++ {
		for x := 6; x < 11; x++ {
			img.Set(x, y, color.NRGBA{G: 200, A: 255})
		}
	}
	box := ContentBounds(img, 8)
	if box != image.Rect(6, 5, 11, 9) {
		t.Fatalf("bounds %v", box)
	}
	trim := TrimTransparent(img, 0)
	if trim.Bounds().Dx() != 5 || trim.Bounds().Dy() != 4 {
		t.Fatalf("trim size %v", trim.Bounds())
	}
}

func TestKnockoutKeepsInteriorHighlights(t *testing.T) {
	t.Parallel()
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// blue ring with a white highlight in the hole
	for y := 8; y < 24; y++ {
		for x := 8; x < 24; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	for y := 13; y < 19; y++ {
		for x := 13; x < 19; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	out := KnockoutBackground(img)
	_, _, _, a0 := out.At(0, 0).RGBA()
	if a0 != 0 {
		t.Fatalf("corner should be transparent, a=%d", a0>>8)
	}
	_, _, _, ac := out.At(16, 16).RGBA()
	if ac < 0x8000 {
		t.Fatalf("interior highlight should stay opaque, a=%d", ac>>8)
	}
}

func TestKnockoutBackgroundLightCanvas(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	// white canvas
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			img.Set(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	// blue blob in center
	for y := 10; y < 22; y++ {
		for x := 10; x < 22; x++ {
			img.Set(x, y, color.NRGBA{B: 200, A: 255})
		}
	}
	out := KnockoutBackground(img)
	_, _, _, a0 := out.At(0, 0).RGBA()
	if a0 != 0 {
		t.Fatalf("corner should be transparent, a=%d", a0>>8)
	}
	_, _, _, ac := out.At(16, 16).RGBA()
	if ac < 0x8000 {
		t.Fatalf("center should stay opaque, a=%d", ac>>8)
	}
}
