package brightness

import "testing"

func TestClamp01(t *testing.T) {
	if clamp01(-0.2) != 0 || clamp01(1.5) != 1 || clamp01(0.25) != 0.25 {
		t.Fatal("clamp01")
	}
}
