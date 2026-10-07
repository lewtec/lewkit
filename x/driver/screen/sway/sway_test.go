package sway

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
)

func TestDPMSOn(t *testing.T) {
	if !dpmsOn(`{"dpms": true}`) || dpmsOn(`{"dpms": false}`) {
		t.Fatal("dpms")
	}
}

func TestMissingWayland(t *testing.T) {
	ctx := driver.WithEnv(t.Context(), []string{"WAYLAND_DISPLAY="})
	err := factory{}.CheckCompatibility(ctx)
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatalf("got %v", err)
	}
}
