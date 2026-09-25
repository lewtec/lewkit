package pulse

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver"
)

func TestParseVolume(t *testing.T) {
	got, err := parseVolume("Volume: front-left: 26214 /  40% / -23.88 dB")
	if err != nil || got != 0.4 {
		t.Fatalf("got %v %v", got, err)
	}
	got, err = parseVolume("no percent here")
	if err != nil || got != 0 {
		t.Fatalf("missing percent: %v %v", got, err)
	}
	if _, err := parseVolume("xx%"); err == nil {
		t.Fatal("expected atoi error")
	}
}

func TestMissingBinary(t *testing.T) {
	err := requireBinary("lewkit-missing-pactl")
	if !errors.Is(err, driver.ErrIncompatible) {
		t.Fatalf("got %v", err)
	}
}
