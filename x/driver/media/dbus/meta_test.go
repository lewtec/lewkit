package dbus

import (
	"testing"

	"github.com/godbus/dbus/v5"
	"github.com/lewtec/lewkit/x/driver/media"
)

func TestPickPlayer(t *testing.T) {
	best, ok := pickPlayer([]playerInfo{
		{name: "a", status: "Paused"},
		{name: "b", status: "Playing"},
		{name: "c", status: "Stopped"},
	})
	if !ok || best.name != "b" {
		t.Fatalf("%+v %v", best, ok)
	}
	if _, ok := pickPlayer(nil); ok {
		t.Fatal("empty")
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
		t.Fatalf("%+v", got)
	}
	if artistOf([]any{"C", 1, "D"}) != "C, D" {
		t.Fatal("artist any")
	}
}
