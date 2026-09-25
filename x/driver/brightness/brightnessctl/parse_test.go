package brightnessctl

import (
	"errors"
	"testing"

	"github.com/lewtec/lewkit/x/driver/brightness"
)

func TestParseStatus(t *testing.T) {
	dev, err := parseStatus("intel_backlight,backlight,50,25%,200\n")
	if err != nil {
		t.Fatal(err)
	}
	if dev.Name != "intel_backlight" || dev.Brightness != 0.25 {
		t.Fatalf("%+v", dev)
	}
	_, err = parseStatus("short\nbad,class,1,nope%,2\n")
	if !errors.Is(err, brightness.ErrDeviceNotFound) {
		t.Fatalf("got %v", err)
	}
}
