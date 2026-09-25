package sway

import (
	"context"
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
	ctx := driver.WithEnv(context.Background(), []string{"WAYLAND_DISPLAY="})
	err := factory{}.CheckCompatibility(ctx)
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatalf("got %v", err)
	}
}
