// Package icons generates multi-platform app icon trees from one master image.
package icons

import (
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Sentinel errors for icon generation.
var (
	ErrSVGNotRasterized    = errors.New("svg masters are not rasterized in-process yet; pass a PNG/JPEG, or convert first (tool catalog TBD)")
	ErrAndroidIconsMissing = errors.New("android icons missing (run build icons)")
	ErrMacOSIconMissing    = errors.New("macos icon.icns missing (run build icons)")
)

// Options drives Generate.
type Options struct {
	// SourcePath is a PNG/JPEG master. Empty uses the embedded default mark.
	SourcePath string
	// OutputDir is the tree root (default: dist/icons relative to caller).
	OutputDir string
	// Force regenerates even when Complete(OutputDir).
	Force bool
}

// Manifest is written to manifest.json.
type Manifest struct {
	GeneratedAt string            `json:"generated_at"`
	Source      string            `json:"source"` // "default" or path
	OutputDir   string            `json:"output_dir"`
	Files       map[string]string `json:"files"` // relative path → note
}

// DefaultOutputDir is the SPEC default tree root name.
const DefaultOutputDir = "dist/icons"

// defaultMaster crops the square mark from the vendored lockup.
func defaultMaster() (image.Image, error) {
	img, err := DecodeBytes(DefaultLockupPNG, "default/lockup.png")
	if err != nil {
		return nil, fmt.Errorf("default lockup: %w", err)
	}
	return Resize(ExtractUpperMark(KnockoutBackground(img)), 1024), nil
}

// Generate writes a full icon matrix under opts.OutputDir.
// If the tree is already Complete and !Force, it is a no-op.
func Generate(opts Options) (*Manifest, error) {
	out := strings.TrimSpace(opts.OutputDir)
	if out == "" {
		out = DefaultOutputDir
	}
	absOut, err := filepath.Abs(out)
	if err != nil {
		return nil, err
	}

	if !opts.Force && Complete(absOut) {
		m, err := readManifest(absOut)
		if err == nil {
			return m, nil
		}
		// incomplete manifest but files present — rebuild
	}

	srcLabel := "default"
	var img image.Image
	if p := strings.TrimSpace(opts.SourcePath); p != "" {
		srcLabel = p
		img, err = DecodeImage(p)
		if err != nil {
			return nil, err
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".svg" {
			return nil, ErrSVGNotRasterized
		}
	} else {
		img, err = defaultMaster()
		if err != nil {
			return nil, err
		}
	}

	// Knock out light photo canvas so splash/launcher icons are not boxed.
	img = KnockoutBackground(img)
	square := PadCenter(img)
	// Work from a high-res square for downscales
	if square.Bounds().Dx() < 1024 {
		square = Resize(square, 1024)
	} else if square.Bounds().Dx() > 1024 {
		square = Resize(square, 1024)
	}

	if err := os.MkdirAll(absOut, 0o755); err != nil {
		return nil, err
	}

	files := map[string]string{}
	record := func(rel, note string) { files[rel] = note }

	// source/
	masterPath := filepath.Join(absOut, "source", "master.png")
	if err := WritePNG(masterPath, square); err != nil {
		return nil, err
	}
	record("source/master.png", "normalized square master")

	// windows/
	winPath := filepath.Join(absOut, "windows", "icon.ico")
	if err := WriteICO(winPath, square, WindowsICOSizes); err != nil {
		return nil, err
	}
	record("windows/icon.ico", "multi-size ico")

	// macos/
	macPath := filepath.Join(absOut, "macos", "icon.icns")
	if err := WriteICNS(macPath, square, MacOSICNSSizes); err != nil {
		return nil, err
	}
	record("macos/icon.icns", "png-in-icns")

	// linux/
	for _, s := range LinuxPNGSizes {
		rel := fmt.Sprintf("linux/icon-%d.png", s)
		if err := WritePNG(filepath.Join(absOut, rel), Resize(square, s)); err != nil {
			return nil, err
		}
		record(rel, fmt.Sprintf("%dpx", s))
	}

	// android/
	for _, m := range AndroidMipmaps {
		rel := filepath.Join("android", m.Dir, "ic_launcher.png")
		if err := WritePNG(filepath.Join(absOut, rel), Resize(square, m.Size)); err != nil {
			return nil, err
		}
		record(filepath.ToSlash(rel), m.Dir)
	}

	// web/
	for _, w := range WebPNGSizes {
		rel := filepath.Join("web", w.Name)
		if err := WritePNG(filepath.Join(absOut, rel), Resize(square, w.Size)); err != nil {
			return nil, err
		}
		record(filepath.ToSlash(rel), fmt.Sprintf("%dpx", w.Size))
	}
	webIco := filepath.Join(absOut, "web", "favicon.ico")
	if err := WriteICO(webIco, square, []int{16, 32, 48}); err != nil {
		return nil, err
	}
	record("web/favicon.ico", "16/32/48")

	man := &Manifest{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Source:      srcLabel,
		OutputDir:   absOut,
		Files:       files,
	}
	raw, err := json.MarshalIndent(man, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(absOut, "manifest.json"), append(raw, '\n'), 0o644); err != nil {
		return nil, err
	}
	return man, nil
}

// Complete reports whether all ExpectedRelative files exist under dir.
func Complete(dir string) bool {
	for _, rel := range ExpectedRelative() {
		if st, err := os.Stat(filepath.Join(dir, rel)); err != nil || st.IsDir() {
			return false
		}
	}
	return true
}

// ApplyAndroidRes copies android/* mipmaps into an Android res/ directory
// (e.g. app/src/main/res) and is a no-op if iconRoot android tree is missing.
func ApplyAndroidRes(iconRoot, androidResDir string) error {
	srcRoot := filepath.Join(iconRoot, "android")
	if st, err := os.Stat(srcRoot); err != nil || !st.IsDir() {
		return fmt.Errorf("%w under %s", ErrAndroidIconsMissing, iconRoot)
	}
	for _, m := range AndroidMipmaps {
		src := filepath.Join(srcRoot, m.Dir, "ic_launcher.png")
		dstDir := filepath.Join(androidResDir, m.Dir)
		if err := os.MkdirAll(dstDir, 0o755); err != nil {
			return err
		}
		dst := filepath.Join(dstDir, "ic_launcher.png")
		raw, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(dst, raw, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// ApplyMacOSICNS copies macos/icon.icns to destICNS (full file path).
func ApplyMacOSICNS(iconRoot, destICNS string) error {
	src := filepath.Join(iconRoot, "macos", "icon.icns")
	raw, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrMacOSIconMissing, err)
	}
	if err := os.MkdirAll(filepath.Dir(destICNS), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destICNS, raw, 0o644)
}

func readManifest(dir string) (*Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return &m, nil
}
