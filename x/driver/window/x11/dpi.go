//go:build linux && !android

package x11

import (
	"math"
	"strconv"
	"strings"

	"github.com/jezek/xgb"
	"github.com/jezek/xgb/xproto"
	"github.com/lewtec/lewkit/x/driver/window"
)

// displayScale reads Xft.dpi from the root RESOURCE_MANAGER property.
// A missing or unusable value is one, and the window stays at Config size.
func displayScale(conn *xgb.Conn) window.Scale {
	if conn == nil {
		return 1
	}
	screen := xproto.Setup(conn).DefaultScreen(conn)
	reply, err := xproto.GetProperty(conn, false, screen.Root, xproto.AtomResourceManager,
		xproto.AtomString, 0, 1<<16).Reply()
	if err != nil || reply == nil || len(reply.Value) == 0 {
		return 1
	}
	dpi := parseXftDPI(string(reply.Value))
	if dpi <= 0 {
		return 1
	}
	return window.Scale(dpi / float64(window.BaselineDPI))
}

// parseXftDPI returns the Xft.dpi number, or 0 when the resource is absent
// or not a positive finite value. Lines are X resource specs. A "!" line is a comment.
func parseXftDPI(raw string) float64 {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "!") {
			continue
		}
		key, val, ok := strings.Cut(line, ":")
		if !ok || strings.TrimSpace(key) != "Xft.dpi" {
			continue
		}
		f, err := strconv.ParseFloat(strings.TrimSpace(val), 64)
		if err != nil || f <= 0 || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0
		}
		return f
	}
	return 0
}

func clampDevice(n int) int {
	if n > 0xFFFF {
		return 0xFFFF
	}
	if n < 1 {
		return 1
	}
	return n
}
