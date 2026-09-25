package x11

import (
	"context"
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
)

func TestMonitorOn(t *testing.T) {
	if !monitorOn("Monitor is On") || monitorOn("Monitor is Off") {
		t.Fatal("monitor")
	}
}

func TestMissingDisplay(t *testing.T) {
	ctx := driver.WithEnv(context.Background(), []string{"DISPLAY="})
	err := factory{}.CheckCompatibility(ctx)
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatalf("got %v", err)
	}
}
