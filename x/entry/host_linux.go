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
	if !linuxApp() {
		return
	}
	id := markerID()
	if id == "" {
		stamped, err := release.AppID()
		if err != nil {
			return
		}
		id = stamped
	}
	gtk.SetPrgname(id)
}

func windowsGUI() bool { return false }

// linuxApp reports whether this executable sits in a lewkit desktop bundle.
func linuxApp() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	_, err = os.Stat(filepath.Join(filepath.Dir(exe), release.MarkerFile))
	return err == nil
}

func markerID() string {
	exe, err := os.Executable()
	if err != nil {
		return ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	raw, err := os.ReadFile(filepath.Join(filepath.Dir(exe), release.MarkerFile))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}
