//go:build linux

package window

import (
	"bytes"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
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

// ApplyXIcon sets _NET_WM_ICON on an existing X11 window.
// A missing display or an empty icon leaves the window unchanged.
func ApplyXIcon(xid uint32, icon image.Image) error {
	words := ShellPicture(icon)
	if xid == 0 || len(words) == 0 || os.Getenv("DISPLAY") == "" {
		return nil
	}
	conn, err := xgb.NewConn()
	if err != nil {
		return err
	}
	defer conn.Close()
	name := "_NET_WM_ICON"
	atom, err := xproto.InternAtom(conn, false, uint16(len(name)), name).Reply()
	if err != nil {
		return err
	}
	raw := make([]byte, len(words)*4)
	for i, v := range words {
		raw[i*4] = byte(v)
		raw[i*4+1] = byte(v >> 8)
		raw[i*4+2] = byte(v >> 16)
		raw[i*4+3] = byte(v >> 24)
	}
	return xproto.ChangePropertyChecked(conn, xproto.PropModeReplace, xproto.Window(xid),
		atom.Atom, xproto.AtomCardinal, 32, uint32(len(words)), raw).Check()
}
