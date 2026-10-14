//go:build linux

package kms

import (
	"testing"
	"time"

	"github.com/lewtec/lewkit/x/driver/window"
	"github.com/stretchr/testify/require"
)

func TestInputKeyAndTablet(t *testing.T) {
	buf := window.NewBuffer(200, 100)
	w := &fbWin{Buffer: buf, width: 200, height: 100}
	events := buf.Subscribe(t.Context())
	dev := &evDev{minX: 0, maxX: 32767, minY: 0, maxY: 32767, abs: true}

	w.apply(dev, evKey, 42, 1)
	w.apply(dev, evKey, 30, 1)
	var key window.Key
	for key.Rune != 'A' {
		key = recvKey(t, events)
	}
	require.True(t, key.Pressed)
	require.Equal(t, window.ModShift, key.Mod)

	w.apply(dev, evAbs, absX, 32767)
	w.apply(dev, evAbs, absY, 0)
	w.apply(dev, evKey, btnLeft, 1)
	ptr := recvPointer(t, events)
	require.Equal(t, 1, ptr.Button)
	require.True(t, ptr.Pressed)
	require.Equal(t, window.ButtonLeft, ptr.Buttons)
	w.apply(dev, evSyn, synReport, 0)
	ptr = recvPointer(t, events)
	require.Equal(t, 199, ptr.Pos.X)
	require.Equal(t, 0, ptr.Pos.Y)
}

func recvKey(t *testing.T, events <-chan window.Event) window.Key {
	t.Helper()
	select {
	case ev := <-events:
		key, ok := ev.(window.Key)
		require.True(t, ok)
		return key
	case <-time.After(time.Second):
		t.Fatal("no key")
	}
	return window.Key{}
}

func recvPointer(t *testing.T, events <-chan window.Event) window.Pointer {
	t.Helper()
	select {
	case ev := <-events:
		ptr, ok := ev.(window.Pointer)
		require.True(t, ok)
		return ptr
	case <-time.After(time.Second):
		t.Fatal("no pointer")
	}
	return window.Pointer{}
}
