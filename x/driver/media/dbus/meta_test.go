//go:build linux && !android

package dbus

import (
	"fmt"
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver/media"
	"github.com/stretchr/testify/require"
)

func TestPickPlayer(t *testing.T) {
	best, ok := pickPlayer([]playerInfo{
		{name: "a", status: "Paused"},
		{name: "b", status: "Playing"},
		{name: "c", status: "Stopped"},
	})
	if !ok || best.name != "b" {
		require.FailNowf(t, "", "%+v %v", best, ok)
	}
	if _, ok := pickPlayer(nil); ok {
		require.FailNow(t, fmt.Sprint("empty"))
	}
}

func TestMetadataFrom(t *testing.T) {
	fields := map[string]dbus.Variant{
		"xesam:title":  dbus.MakeVariant("Song"),
		"xesam:artist": dbus.MakeVariant([]string{"A", "B"}),
		"mpris:artUrl": dbus.MakeVariant("https://example/art"),
		"mpris:length": dbus.MakeVariant(uint64(5)),
	}
	got := metadataFrom("player", media.StatusPlaying, fields)
	if got.Title != "Song" || got.Artist != "A, B" || got.ArtUrl != "https://example/art" || got.Length != 5 {
		require.FailNowf(t, "", "%+v", got)
	}
	if artistOf([]any{"C", 1, "D"}) != "C, D" {
		require.FailNow(t, fmt.Sprint("artist any"))
	}
}
