package volume

import (
	"testing"

	"github.com/lewtec/lewkit/x/driver"
)

func TestClamp01(t *testing.T) {
	if driver.Clamp01(1+step) != 1 || driver.Clamp01(0-step) != 0 {
		t.Fatalf("step clamp")
	}
}
