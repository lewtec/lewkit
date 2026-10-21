//go:build linux && !android

package x11

import (
	"image"
	"os"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/lewtec/lewkit/x/driver/window"
)

// ApplyXIcon sets _NET_WM_ICON on an existing X11 window.
// A missing display or an empty icon leaves the window unchanged.
func ApplyXIcon(xid uint32, icon image.Image) error {
	words := window.ShellPicture(icon)
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
