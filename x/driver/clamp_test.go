package driver

import "testing"

func TestClamp01(t *testing.T) {
	if Clamp01(-0.2) != 0 || Clamp01(1.5) != 1 || Clamp01(0.25) != 0.25 {
		t.Fatal("clamp01")
	}
}
