//go:build linux

package dbus

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver/tray"
	"github.com/stretchr/testify/require"
)

func TestOpenUpdateClose(t *testing.T) {
	if os.Getenv("DBUS_SESSION_BUS_ADDRESS") == "" {
		t.Skip("no session bus")
	}
	img := image.NewNRGBA(image.Rect(0, 0, 16, 16))
	for y := range 16 {
		for x := range 16 {
			img.SetNRGBA(x, y, color.NRGBA{R: 20, G: 120, B: 80, A: 255})
		}
	}
	clicked := make(chan struct{}, 1)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	item, err := tray.Open(ctx, tray.Config{
		Title: "lewkit tray test",
		Icon:  tray.Icon{Image: img},
		Menu:  []tray.Item{{Label: "Ping", OnClick: func() { clicked <- struct{}{} }}},
	})
	var busErr *dbus.Error
	if errors.As(err, &busErr) {
		t.Skip(err)
	}
	require.NoError(t, err)
	require.NoError(t, callMenu(t, "clicked"))
	require.Eventually(t, func() bool {
		select {
		case <-clicked:
			return true
		default:
			return false
		}
	}, 2*time.Second, 10*time.Millisecond)
	require.NoError(t, item.Update(tray.Config{
		Title:   "lewkit tray test",
		Tooltip: "updated",
		Icon:    tray.Icon{Name: "applications-system"},
		Menu: []tray.Item{
			{Label: "More", Children: []tray.Item{{Label: "About"}}},
			{Separator: true},
			{Label: "Quit"},
		},
	}))
	require.NoError(t, item.Close())
	require.ErrorIs(t, item.Update(tray.Config{Title: "again"}), tray.ErrClosed)
}

var errNotifierMissing = errors.New("status notifier missing")

func callMenu(t *testing.T, event string) error {
	t.Helper()
	conn, err := dbus.ConnectSessionBus()
	if err != nil {
		return err
	}
	defer conn.Close()
	var names []string
	if err := conn.BusObject().Call("org.freedesktop.DBus.ListNames", 0).Store(&names); err != nil {
		return err
	}
	prefix := fmt.Sprintf("org.kde.StatusNotifierItem-%d-", os.Getpid())
	var name string
	for _, candidate := range names {
		if strings.HasPrefix(candidate, prefix) {
			name = candidate
		}
	}
	if name == "" {
		return fmt.Errorf("%w: %s", errNotifierMissing, prefix)
	}
	return conn.Object(name, "/MenuBar").Call(
		"com.canonical.dbusmenu.Event",
		0,
		int32(1),
		event,
		dbus.MakeVariant(int32(0)),
		uint32(0),
	).Err
}
