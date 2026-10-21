//go:build linux && !android

package x11

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseXftDPI(t *testing.T) {
	raw := "! comment\n\nXft.antialias:\t1\nXft.dpi:\t192\nXft.rgba:\tnone\n"
	assert.Equal(t, 192.0, parseXftDPI(raw))
	assert.Equal(t, 96.0, parseXftDPI("Xft.dpi: 96"))
	assert.Equal(t, 0.0, parseXftDPI(""))
	assert.Equal(t, 0.0, parseXftDPI("! Xft.dpi:\t192\n"))
	assert.Equal(t, 0.0, parseXftDPI("Xft.dpi:\tnope"))
	assert.Equal(t, 0.0, parseXftDPI("Xft.dpi:\t-10"))
	assert.Equal(t, 120.5, parseXftDPI("Xft.dpi: 120.5"))
}

func TestClampDevice(t *testing.T) {
	assert.Equal(t, 640, clampDevice(640))
	assert.Equal(t, 0xFFFF, clampDevice(0x10000))
	assert.Equal(t, 1, clampDevice(0))
}
