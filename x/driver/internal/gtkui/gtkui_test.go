package gtkui

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
)

func TestRequireNeedsDisplay(t *testing.T) {
	t.Setenv("DISPLAY", "")
	t.Setenv("WAYLAND_DISPLAY", "")
	err := Require(t.Context())
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatal(err)
	}
	if runtime.GOOS == "linux" && !strings.Contains(err.Error(), "DISPLAY") {
		t.Fatal(err)
	}
}

func TestRunCallsHere(t *testing.T) {
	err := Run(t.Context(), func() error { return errors.New("here") })
	if err == nil || err.Error() != "here" {
		t.Fatal(err)
	}
}
