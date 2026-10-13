//go:build linux && !android

package window

import (
	"bytes"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/release"
)

var (
	bundleOnce sync.Once
	bundleImg  image.Image
)

// BundleIcon is icon.png from this process's AppImage trailer.
// A process that is not a Linux AppImage gets nil.
func BundleIcon() image.Image {
	bundleOnce.Do(func() {
		exe, err := os.Executable()
		if err != nil {
			return
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		tr, err := release.OpenTrailer(exe)
		if err != nil {
			return
		}
		defer tr.Close()
		marker, err := tr.Bytes(release.MarkerFile)
		if err != nil || strings.TrimSpace(string(marker)) == "" {
			return
		}
		raw, err := tr.Bytes("icon.png")
		if err != nil {
			return
		}
		img, _, err := image.Decode(bytes.NewReader(raw))
		if err != nil {
			return
		}
		bundleImg = img
	})
	return bundleImg
}

func readIcon(path string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	img, _, err := image.Decode(file)
	return img, err
}
