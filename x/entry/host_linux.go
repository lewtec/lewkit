//go:build linux

package entry

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/lewtec/lewkit/x/ffi/native"
	"github.com/lewtec/lewkit/x/ffi/native/gtk"
	"github.com/lewtec/lewkit/x/release"
)

func prepareHost(ctx context.Context) {
	if err := native.Prepare(ctx); err != nil {
		return
	}
	id, ok := appMarker()
	if !ok {
		return
	}
	gtk.SetPrgname(id)
}

func windowsGUI() bool { return false }

// linuxApp reports whether this executable is a lewkit AppImage.
func linuxApp() bool {
	_, ok := appMarker()
	return ok
}

func appMarker() (string, bool) {
	exe, err := os.Executable()
	if err != nil {
		return "", false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	tr, err := release.OpenTrailer(exe)
	if err != nil {
		return "", false
	}
	defer tr.Close()
	raw, err := tr.Bytes(release.MarkerFile)
	if err != nil {
		return "", false
	}
	id := strings.TrimSpace(string(raw))
	if id == "" {
		return "", false
	}
	return id, true
}
