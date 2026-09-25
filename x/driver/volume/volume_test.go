package volume

import "testing"

func TestClamp01(t *testing.T) {
	if clamp01(-0.1) != 0 || clamp01(1.2) != 1 || clamp01(0.4) != 0.4 {
		t.Fatalf("clamp01 bounds")
	}
	if clamp01(1+step) != 1 || clamp01(0-step) != 0 {
		t.Fatalf("step clamp")
	}
}
